package api

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/9seconds/topographer/config"
	"github.com/9seconds/topographer/providers"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) *providers.ProviderSet {
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
	set := providers.NewProviderSet(conf)

	err := set.Providers["maxmind"].Reopen(time.Now())
	assert.Nil(t, err)

	return set
}

func TestBuild(t *testing.T) {
	set := setup(t)
	ip := net.ParseIP("89.160.20.112")
	result := set.Resolve([]net.IP{ip}, []string{})
	resp := &ipResolveResponseStruct{}
	resp.Build(result)

	assert.Equal(t, resp.Results["89.160.20.112"].Country, "se")
	assert.Equal(t, resp.Results["89.160.20.112"].Details["maxmind"].Country, "se")
}
