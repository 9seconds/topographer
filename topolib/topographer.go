package topolib

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/antzucaro/matchr"
	"github.com/panjf2000/ants/v2"
)

const (
	DefaultWorkerPoolSize = 4096

	workerPoolExpireTime = time.Minute
)

type Topographer struct {
	logger     Logger
	providers  map[string]Provider
	rwmutex    sync.RWMutex
	closeOnce  sync.Once
	workerPool *ants.PoolWithFunc
	closed     bool
}

func (t *Topographer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	httpHandler{t}.ServeHTTP(w, req)
}

func (t *Topographer) ResolveAll(ctx context.Context,
	ips []net.IP,
	providers []string) ([]ResolveResult, error) {
	t.rwmutex.RLock()
	defer t.rwmutex.RUnlock()

	if t.closed {
		return nil, ErrTopographerShutdown
	}

	providersToUse, err := t.getProvidersToUse(providers)
	if err != nil {
		return nil, err
	}

	resultChannel := make(chan ResolveResult, len(ips))
	rv := make([]ResolveResult, 0, len(ips))
	wg := &sync.WaitGroup{}
	groupRequest := newPoolGroupRequest(ctx, resultChannel,
		providersToUse, wg, t.workerPool)

	ipsToIndex := map[string]int{}

	for i, v := range ips {
		vv := v.To16()

		if err := groupRequest.Do(ctx, vv); err != nil {
			break
		}

		ipsToIndex[string(vv)] = i
	}

	go func() {
		wg.Wait()
		close(resultChannel)
	}()

	for res := range resultChannel {
		rv = append(rv, res)
	}

	sort.Slice(rv, func(i, j int) bool {
		return ipsToIndex[string(rv[i].IP)] < ipsToIndex[string(rv[j].IP)]
	})

	return rv, nil
}

func (t *Topographer) Resolve(ctx context.Context,
	ip net.IP,
	providers []string) (ResolveResult, error) {
	t.rwmutex.RLock()
	defer t.rwmutex.RUnlock()

	ip = ip.To16()
	rv := ResolveResult{
		IP: ip,
	}

	if t.closed {
		return rv, ErrTopographerShutdown
	}

	providersToUse, err := t.getProvidersToUse(providers)
	if err != nil {
		return rv, err
	}

	resultChannel := make(chan ResolveResult)
	wg := &sync.WaitGroup{}
	groupRequest := newPoolGroupRequest(ctx, resultChannel,
		providersToUse, wg, t.workerPool)

	if err := groupRequest.Do(ctx, ip); err != nil {
		return rv, nil
	}

	rv = <-resultChannel

	wg.Wait()
	close(resultChannel)

	return rv, nil
}

func (t *Topographer) getProvidersToUse(names []string) ([]Provider, error) {
	rv := make([]Provider, 0, len(names))

	if len(names) == 0 {
		for _, v := range t.providers {
			rv = append(rv, v)
		}
	} else {
		for _, v := range names {
			vv, ok := t.providers[v]
			if !ok {
				return nil, fmt.Errorf("provider %s is unknown", v)
			}

			rv = append(rv, vv)
		}
	}

	return rv, nil
}

func (t *Topographer) Shutdown() {
	t.rwmutex.Lock()
	defer t.rwmutex.Unlock()

	t.closed = true

	t.closeOnce.Do(func() {
		t.workerPool.Release()

		for _, v := range t.providers {
			if vv, ok := v.(*fsUpdater); ok {
				vv.Shutdown()
			}
		}
	})
}

func (t *Topographer) resolveIP(args interface{}) {
	params := args.(*resolveIPRequest)
	defer params.wg.Done()

	rv := make([]ResolveResultDetail, 0, len(params.providers))
	taskChannel := make(chan ResolveResultDetail, len(params.providers))
	wg := &sync.WaitGroup{}

	wg.Add(len(params.providers))

	go func() {
		wg.Wait()
		close(taskChannel)
	}()

	for _, v := range params.providers {
		go t.resolveIPLookup(params.ctx, params.ip, v, taskChannel, wg)
	}

	for res := range taskChannel {
		rv = append(rv, res)
	}

	select {
	case <-params.ctx.Done():
	case params.resultChannel <- t.resolveIPMerge(params.ip, rv):
	}
}

func (t *Topographer) resolveIPLookup(ctx context.Context,
	ip net.IP,
	provider Provider,
	taskChannel chan<- ResolveResultDetail,
	wg *sync.WaitGroup) {
	defer wg.Done()

	detail := ResolveResultDetail{
		ProviderName: provider.Name(),
	}

	if res, err := provider.Lookup(ctx, ip); err != nil {
		t.logger.LookupError(ip, provider.Name(), err)
	} else {
		detail.City = res.City
		detail.CountryCode = res.CountryCode
	}

	select {
	case <-ctx.Done():
	case taskChannel <- detail:
	}
}

func (t *Topographer) resolveIPMerge(ip net.IP, results []ResolveResultDetail) ResolveResult {
	countries := map[CountryCode][]*ResolveResultDetail{}

	for i := range results {
		current := &results[i]

		if !current.CountryCode.Known() {
			continue
		}

		arr, ok := countries[current.CountryCode]

		if !ok {
			arr = []*ResolveResultDetail{}
			countries[current.CountryCode] = arr
		}

		countries[current.CountryCode] = append(arr, current)
	}

	var cityResults []*ResolveResultDetail
	var selectedCountry CountryCode

	maxLen := 0

	for country, group := range countries {
		if len(group) > maxLen {
			cityResults = group
			selectedCountry = country
			maxLen = len(group)
		}
	}

	rv := ResolveResult{
		IP:      ip.To16(),
		Details: results,
		City:    t.resolveIPMergeCity(cityResults),
	}

    if selectedCountry.Known() {
        details := selectedCountry.Details()
		rv.Country.Alpha2Code = details.Alpha2
		rv.Country.Alpha3Code = details.Alpha3
		rv.Country.CommonName = details.Name.Common
		rv.Country.OfficialName = details.Name.Official
    }

	return rv
}

func (t *Topographer) resolveIPMergeCity(results []*ResolveResultDetail) string {
	counters := make(map[string]int)
	names := make(map[string]string)

	for _, v := range results {
		if v.City == "" {
			continue
		}

		normalizedCityName, _ := matchr.DoubleMetaphone(v.City)

		counters[normalizedCityName] += 1
		names[normalizedCityName] = v.City
	}

	maxLen := 0
	cityName := ""

	for k, v := range counters {
		if v > maxLen {
			cityName = names[k]
			maxLen = v
		}
	}

	return cityName
}

func NewTopographer(providers []Provider, logger Logger, workerPoolSize int) (*Topographer, error) {
	rv := &Topographer{
		logger:    logger,
		providers: map[string]Provider{},
	}

	for _, v := range providers {
		if vv, ok := v.(OfflineProvider); ok {
			ctx, cancel := context.WithCancel(context.Background())
			updater := &fsUpdater{
				ctx:      ctx,
				cancel:   cancel,
				logger:   logger,
				provider: vv,
			}

			if err := updater.Start(); err != nil {
				return nil, fmt.Errorf("cannot start provider %s: %w", v.Name(), err)
			}

			v = updater
		}

		rv.providers[v.Name()] = v
	}

	poolSize := workerPoolSize
	if poolSize <= 0 {
		poolSize = DefaultWorkerPoolSize
	}

	rv.workerPool, _ = ants.NewPoolWithFunc(poolSize, rv.resolveIP,
		ants.WithExpiryDuration(workerPoolExpireTime))

	return rv, nil
}
