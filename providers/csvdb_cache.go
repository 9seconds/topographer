package providers

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
)

const csvdbCacheSize = 3092

var globalCSVDBCache csvdbCache

type cacheIface interface {
	Add(key, value interface{}) bool
	Get(key interface{}) (interface{}, bool)
}

type csvdbCache struct {
	size  int
	data  map[string]cacheIface
	mutex *sync.Mutex
}

func (cc *csvdbCache) get(country, city string) *GeoResult {
	cache := cc.getCache(country)
	if item, ok := cache.Get(city); ok {
		return item.(*GeoResult)
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	if item, ok := cache.Get(city); ok {
		return item.(*GeoResult)
	}

	item := &GeoResult{Country: country, City: city}
	cache.Add(city, item)

	return item
}

func (cc *csvdbCache) getCache(country string) cacheIface {
	if got, ok := cc.data[country]; ok {
		return got
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	if got, ok := cc.data[country]; ok {
		return got
	}
	newCache, _ := lru.New(cc.size)
	cc.data[country] = newCache

	return newCache
}

func newCSVDBCache(size int) csvdbCache {
	return csvdbCache{
		size:  size,
		data:  make(map[string]cacheIface),
		mutex: &sync.Mutex{},
	}
}

func init() {
	globalCSVDBCache = newCSVDBCache(csvdbCacheSize)
}
