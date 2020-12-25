package topolib

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/antzucaro/matchr"
	"github.com/panjf2000/ants/v2"
	"github.com/pariz/gountries"
)

const DefaultWorkerPoolSize = 4096

type Topographer struct {
	logger     Logger
	providers  map[string]Provider
	rwmutex    sync.RWMutex
	closeOnce  sync.Once
	countries  *gountries.Query
	workerPool *ants.PoolWithFunc
	closed     bool
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

	for _, v := range ips {
		if err := groupRequest.Do(ctx, v); err != nil {
			break
		}
	}

	go func() {
		wg.Wait()
		close(resultChannel)
	}()

	for res := range resultChannel {
		rv = append(rv, res)
	}

	return rv, nil
}

func (t *Topographer) Resolve(ctx context.Context,
	ip net.IP,
	providers []string) (ResolveResult, error) {
	t.rwmutex.RLock()
	defer t.rwmutex.RUnlock()

	if t.closed {
		return ResolveResult{}, ErrTopographerShutdown
	}

	providersToUse, err := t.getProvidersToUse(providers)
	if err != nil {
		return ResolveResult{}, err
	}

	resultChannel := make(chan ResolveResult)
	wg := &sync.WaitGroup{}
	rv := ResolveResult{}
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
		detail.CountryCode = strings.ToUpper(res.CountryCode)
	}

	select {
	case <-ctx.Done():
	case taskChannel <- detail:
	}
}

func (t *Topographer) resolveIPMerge(ip net.IP, results []ResolveResultDetail) ResolveResult {
	countries := map[string][]*ResolveResultDetail{}

	for i := range results {
		current := &results[i]

		if current.CountryCode == "" {
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

	maxLen := 0
	selectedCountry := ""

	for country, group := range countries {
		if len(group) > maxLen {
			cityResults = group
			selectedCountry = country
			maxLen = len(group)
		}
	}

	rv := ResolveResult{
		IP:      ip,
		Details: results,
		City:    t.resolveIPMergeCity(cityResults),
	}

	if country, err := t.countries.FindCountryByAlpha(selectedCountry); err == nil {
		rv.Country.Alpha2Code = country.Alpha2
		rv.Country.Alpha3Code = country.Alpha3
		rv.Country.CommonName = country.Name.Common
		rv.Country.OfficialName = country.Name.Official
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
		countries: gountries.New(),
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
		ants.WithExpiryDuration(time.Minute))

	return rv, nil
}
