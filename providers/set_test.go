package providers

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/9seconds/topographer/config"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) *ProviderSet {
	wd, _ := os.Getwd()
	testdata := filepath.Join(filepath.Dir(wd), "testdata")
	text := fmt.Sprintf("directory = \"%s\"", testdata)
	text = text + `
	[databases]
		[databases.maxmind]
		enabled = true
		weight = 1.3
	`
	conf, _ := config.Parse(strings.NewReader(text))
	set := NewProviderSet(conf)

	err := set.Providers["maxmind"].Reopen(time.Now())
	assert.Nil(t, err)

	return set
}

func TestResolveEmpty(t *testing.T) {
	set := setup(t)
	ip := net.ParseIP("89.160.20.112")
	assert.Len(t, set.Resolve([]net.IP{ip}, []string{"xxx"}), 0)
}

func TestResolveExplicitSet(t *testing.T) {
	set := setup(t)
	ip := net.ParseIP("89.160.20.112")
	result := set.Resolve([]net.IP{ip}, []string{"maxmind"})

	assert.Len(t, result, 1)
	assert.Len(t, result[0].Results, 1)
	assert.Equal(t, result[0].Provider, "maxmind")
	assert.InDelta(t, result[0].Weight, 1.3, 1e-6)
	assert.Equal(t, result[0].Results["89.160.20.112"].Country, "se")
}

func TestResolveImplicitSet(t *testing.T) {
	set := setup(t)
	ip := net.ParseIP("89.160.20.112")
	result := set.Resolve([]net.IP{ip}, []string{})

	assert.Len(t, result, 1)
	assert.Len(t, result[0].Results, 1)
	assert.Equal(t, result[0].Provider, "maxmind")
	assert.InDelta(t, result[0].Weight, 1.3, 1e-6)
	assert.Equal(t, result[0].Results["89.160.20.112"].Country, "se")
}
