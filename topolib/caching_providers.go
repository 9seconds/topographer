package topolib

import (
	"context"
	"net"
	"time"

	"github.com/dgraph-io/ristretto"
)

type cachingProvider struct {
	Provider

	cache *ristretto.Cache
	ttl   time.Duration
}

func (c cachingProvider) Lookup(ctx context.Context, ip net.IP) (ProviderLookupResult, error) {
	cacheKey := ip.String()

	value, ok := c.cache.Get(cacheKey)
	if ok {
		return value.(ProviderLookupResult), nil
	}

	result, err := c.Provider.Lookup(ctx, ip)
	if err != nil {
		return ProviderLookupResult{}, err
	}

	c.cache.SetWithTTL(cacheKey, result, 1, c.ttl)

	return result, nil
}

type cachingOfflineProvider struct {
	OfflineProvider
	cachingProvider
}

func (c cachingOfflineProvider) Lookup(ctx context.Context, ip net.IP) (ProviderLookupResult, error) {
    return c.cachingProvider.Lookup(ctx, ip)
}

func NewCachingProvider(provider Provider, itemsCount uint, ttl time.Duration) Provider {
	cacheConfig := &ristretto.Config{
		MaxCost:     int64(itemsCount),
		NumCounters: 10 * int64(itemsCount),
		Metrics:     false,
		BufferItems: 64,
	}

	cache, err := ristretto.NewCache(cacheConfig)
	if err != nil {
		panic(err)
	}

	return cachingProvider{
		Provider: provider,
		cache:    cache,
		ttl:      ttl,
	}
}

func NewCachingOfflineProvider(provider OfflineProvider, itemsCount uint, ttl time.Duration) OfflineProvider {
    return cachingOfflineProvider{
        OfflineProvider: provider,
        cachingProvider: NewCachingProvider(provider, itemsCount, ttl).(cachingProvider),
    }
}
