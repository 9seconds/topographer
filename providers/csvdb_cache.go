package providers

import lru "github.com/hashicorp/golang-lru"

type cacheIface interface {
	Add(key, value interface{}) bool
	Get(key interface{}) (interface{}, bool)
}

type csvdbCache struct {
	size int
	data map[string]cacheIface
}

func (cc *csvdbCache) get(country, city string) *GeoResult {
	var cache cacheIface
	if got, ok := cc.data[country]; ok {
		cache = got
	} else {
		newCache, _ := lru.New(cc.size)
		cache = newCache
		cc.data[country] = cache
	}

	if item, ok := cache.Get(city); ok {
		return item.(*GeoResult)
	}

	item := &GeoResult{Country: country, City: city}
	cache.Add(city, item)

	return item
}

func newCSVDBCache(size int) csvdbCache {
	return csvdbCache{
		size: size,
		data: make(map[string]cacheIface),
	}
}
