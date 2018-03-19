package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSVDBCacheDifferentParameters(t *testing.T) {
	cache := newCSVDBCache(128)
	data1 := cache.get("c", "p")
	data2 := cache.get("c", "q")

	assert.Equal(t, data1.Country, data2.Country)
	assert.NotEqual(t, data1.City, data2.City)
	assert.NotEqual(t, data1, data2)
}

func TestCSVDBCacheSameParameters(t *testing.T) {
	cache := newCSVDBCache(128)
	data1 := cache.get("c", "p")
	data2 := cache.get("c", "p")
	data3 := cache.get("c", "p")

	assert.Equal(t, data1, data2)
	assert.Equal(t, data2, data3)
}
