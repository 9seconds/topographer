package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hjson/hjson-go"
)

const (
	DefaultHTTPTimeout       = 10 * time.Second
	DefaultUpdateEvery       = 24 * time.Hour
	DefaultRateLimitInterval = 100 * time.Millisecond
	DefaultRateLimitBurst    = 10
)

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalJSON(b []byte) error {
	var v interface{}

	if err := json.Unmarshal(b, &v); err != nil {
		return fmt.Errorf("cannot unmarshal duration: %w", err)
	}

	vv, ok := v.(string)
	if !ok {
		return fmt.Errorf("incorrect duration: %v", v)
	}

	dur, err := time.ParseDuration(vv)
	if err != nil {
		return fmt.Errorf("cannot parse duration: %w", err)
	}

	d.Duration = dur

	return nil
}

type config struct {
	Listen         string           `json:"listen"`
	RootDirectory  string           `json:"root_directory"`
	WorkerPoolSize uint             `json:"worker_pool_size"`
	Providers      []configProvider `json:"providers"`
}

func (c config) GetListen() string {
	return c.Listen
}

func (c config) GetRootDirectory() string {
	if c.RootDirectory != "" {
		return c.RootDirectory
	}

	return filepath.Join(os.TempDir(), "topographer")
}

func (c config) GetWorkerPoolSize() int {
	return int(c.WorkerPoolSize)
}

func (c config) GetProviders() []configProvider {
	return c.Providers
}

type configProvider struct {
	Name               string            `json:"name"`
	Directory          string            `json:"directory"`
	RateLimitInterval  duration          `json:"rate_limit_interval"`
	RateLimitBurst     uint              `json:"rate_limit_burst"`
	UpdateEvery        duration          `json:"update_every"`
	HTTPTimeout        duration          `json:"http_timeout"`
	SpecificParameters map[string]string `json:"specific_parameters"`
}

func (c configProvider) GetName() string {
	return c.Name
}

func (c configProvider) GetDirectory() string {
	if c.Directory != "" {
		return c.Directory
	}

	return c.Name
}

func (c configProvider) GetRateLimitInterval() time.Duration {
	if c.RateLimitInterval.Duration == 0 {
		return DefaultRateLimitInterval
	}

	return c.RateLimitInterval.Duration
}

func (c configProvider) GetRateLimitBurst() int {
	if c.RateLimitBurst == 0 {
		return DefaultRateLimitBurst
	}

	return int(c.RateLimitBurst)
}

func (c configProvider) GetUpdateEvery() time.Duration {
	if c.UpdateEvery.Duration == 0 {
		return DefaultUpdateEvery
	}

	return c.UpdateEvery.Duration
}

func (c configProvider) GetHTTPTimeout() time.Duration {
	if c.HTTPTimeout.Duration == 0 {
		return DefaultHTTPTimeout
	}

	return c.HTTPTimeout.Duration
}

func (c configProvider) GetSpecificParameters() map[string]string {
	if c.SpecificParameters == nil {
		return map[string]string{}
	}

	return c.SpecificParameters
}

func parseConfig(path string) (*config, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %w", err)
	}

	conf := config{}
	rawMap := map[string]interface{}{}

	if err := hjson.Unmarshal(content, &rawMap); err != nil {
		return nil, fmt.Errorf("cannot parse json: %w", err)
	}

	rawBytes, _ := json.Marshal(rawMap)

    json.Unmarshal(rawBytes, &conf) // nolint: errcheck

	if _, _, err := net.SplitHostPort(conf.Listen); err != nil {
		return nil, fmt.Errorf("incorrect host:port for listen: %w", err)
	}

	conf.RootDirectory, err = filepath.Abs(conf.GetRootDirectory())
	if err != nil {
		return nil, fmt.Errorf("incorrect root directory: %w", err)
	}

	seenProviderNames := map[string]struct{}{}
	seenDirectories := map[string]struct{}{}

	for _, v := range conf.Providers {
		if _, ok := seenProviderNames[v.GetName()]; ok {
			return nil, fmt.Errorf("Name %s is duplicated", v.GetName())
		}

		seenProviderNames[v.GetName()] = struct{}{}

		if _, ok := seenDirectories[v.GetDirectory()]; ok {
			return nil, fmt.Errorf("Directory %s is duplicated", v.GetDirectory())
		}

		seenDirectories[v.GetDirectory()] = struct{}{}
	}

	return &conf, nil
}
