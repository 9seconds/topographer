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
	// A size of the worker pool to use in Topographer.
	//
	// Topographer is using worker pool to access providers. It is
	// done to prevent overloading and overusing of them especially if
	// provider accesses some external resource.
	//
	// A worker task is a single IP lookup for _all_ providers. So, if
	// you have a worker pool size of 100, you can concurrenly resolve
	// only 100 IPs. This is not quite a granular but useful on practice
	// if you plan capacity.
	//
	// Usually you want to have this number of workers in pool.
	DefaultWorkerPoolSize = 4096

	workerPoolExpireTime = time.Minute
)

// Topographer is a main entity of topolib. It is responsible for
// provider management, background updates and IP lookups. It also
// contains an instance of worker pool to use.
type Topographer struct {
	logger        Logger
	providers     map[string]Provider
	providerStats map[string]*UsageStats
	rwmutex       sync.RWMutex
	closeOnce     sync.Once
	workerPool    *ants.PoolWithFunc
	closed        bool
}

// ServeHTTP is to conform http.Handler interface.
func (t *Topographer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	httpHandler{t}.ServeHTTP(w, req)
}

// ResolveAll concurrently resolves IP geolocation of the batch of ip
// addresses.
//
// 'providers' argument contains names of the providers to use. If you
// want to use all providers, simply pass nil here.
func (t *Topographer) ResolveAll(ctx context.Context,
	ips []net.IP,
	providers []string) ([]ResolveResult, error) {
	t.rwmutex.RLock()
	defer t.rwmutex.RUnlock()

	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

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

// Resolve geolocation of the single IP.
//
// 'providers' argument contains names of the providers to use. If you
// want to use all providers, simply pass nil here.
func (t *Topographer) Resolve(ctx context.Context,
	ip net.IP,
	providers []string) (ResolveResult, error) {
	t.rwmutex.RLock()
	defer t.rwmutex.RUnlock()

	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

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

// UsageStats returns an array with stats.
func (t *Topographer) UsageStats() []*UsageStats {
	stats := make([]*UsageStats, 0, len(t.providerStats))

	for _, v := range t.providerStats {
		stats = append(stats, v)
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Name < stats[j].Name
	})

	return stats
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
			if vv, ok := v.(OfflineProvider); ok {
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
	stat := t.providerStats[provider.Name()]

	if res, err := provider.Lookup(ctx, ip); err != nil {
		stat.notifyUsed(err)
		t.logger.LookupError(ip, provider.Name(), err)
	} else {
		detail.City = res.City
		detail.CountryCode = res.CountryCode
		stat.notifyUsed(nil)
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

// NewTopographer creates a new instance of topographer.
func NewTopographer(providers []Provider, logger Logger, workerPoolSize int) (*Topographer, error) {
	rv := &Topographer{
		logger:        logger,
		providers:     map[string]Provider{},
		providerStats: map[string]*UsageStats{},
	}

	poolSize := workerPoolSize
	if poolSize <= 0 {
		poolSize = DefaultWorkerPoolSize
	}

	rv.workerPool, _ = ants.NewPoolWithFunc(poolSize, rv.resolveIP,
		ants.WithExpiryDuration(workerPoolExpireTime))

	for _, v := range providers {
		stat := &UsageStats{
			Name: v.Name(),
		}
		rv.providerStats[v.Name()] = stat

		if vv, ok := v.(OfflineProvider); ok {
			updater, err := newFsUpdater(vv, logger, stat)
			if err != nil {
				rv.Shutdown()

				return nil, fmt.Errorf("cannot start provider %s: %w", v.Name(), err)
			}

			v = updater
		}

		rv.providers[v.Name()] = v
	}

	return rv, nil
}
