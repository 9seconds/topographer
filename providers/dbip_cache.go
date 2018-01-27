package providers

import lru "github.com/hashicorp/golang-lru"

type cacheIface interface {
	Add(key, value interface{}) bool
	Get(key interface{}) (interface{}, bool)
}

type dbipCache struct {
	size int
	data map[string]cacheIface
}

func (dc *dbipCache) get(country, city string) *GeoResult {
	var cache cacheIface
	if got, ok := dc.data[country]; ok {
		cache = got
	} else {
		newCache, _ := lru.New(dc.size)
		cache = newCache
		dc.data[country] = cache
	}

	if item, ok := cache.Get(city); ok {
		return item.(*GeoResult)
	}

	item := &GeoResult{Country: country, City: city}
	cache.Add(city, item)

	return item
}

func newDBIPCache(size int) dbipCache {
	return dbipCache{
		size: size,
		data: make(map[string]cacheIface),
	}
}
