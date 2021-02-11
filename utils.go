package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/9seconds/topographer/providers"
	"github.com/9seconds/topographer/topolib"
)

func makeRootContext() (context.Context, context.CancelFunc) {
	rootCtx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)

	go func() {
		for range sigChan {
			cancel()
		}
	}()

	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	return rootCtx, cancel
}

func makeProviders(conf *config) ([]topolib.Provider, error) {
	rv := make([]topolib.Provider, 0, len(conf.GetProviders()))

	for _, v := range conf.GetProviders() {
		httpClient := makeNewHTTPClient(v)

		switch v.GetName() {
		case providers.NameDBIPLite:
			baseDir, err := ensureDir(conf, v)
			if err != nil {
				return nil, fmt.Errorf("cannot create base directory for dbip provider: %w", err)
			}

			rv = append(rv, providers.NewDBIPLite(httpClient, v.GetUpdateEvery(), baseDir))
		case providers.NameIP2C:
			rv = append(rv, providers.NewIP2C(httpClient))
		case providers.NameIP2Location:
			baseDir, err := ensureDir(conf, v)
			if err != nil {
				return nil, fmt.Errorf("cannot create base directory for ip2location provider: %w", err)
			}

			params := v.GetSpecificParameters()

			prov, err := providers.NewIP2Location(httpClient, v.GetUpdateEvery(), baseDir, params["auth_token"], params["db_code"])
			if err != nil {
				return nil, fmt.Errorf("cannot create ip2location provider: %w", err)
			}

			rv = append(rv, prov)
		case providers.NameIPInfo:
			token := v.GetSpecificParameters()["auth_token"]
			rv = append(rv, providers.NewIPInfo(httpClient, token))
		case providers.NameIPStack:
			params := v.GetSpecificParameters()

			prov, err := providers.NewIPStack(httpClient,
				params["auth_token"], boolParam(params["secure"]))
			if err != nil {
				return nil, fmt.Errorf("cannot create ipstack provider: %w", err)
			}

			rv = append(rv, prov)
		case providers.NameMaxmindLite:
			baseDir, err := ensureDir(conf, v)
			if err != nil {
				return nil, fmt.Errorf("cannot create base directory for maxmind provider: %w", err)
			}

			licenseKey := v.GetSpecificParameters()["license_key"]

			prov, err := providers.NewMaxmindLite(httpClient, v.GetUpdateEvery(), baseDir,
				licenseKey)
			if err != nil {
				return nil, fmt.Errorf("cannot create ipstack provider: %w", err)
			}

			rv = append(rv, prov)
		case providers.NameSoftware77:
			baseDir, err := ensureDir(conf, v)
			if err != nil {
				return nil, fmt.Errorf("cannot create base directory for sofware77 provider: %w", err)
			}

			rv = append(rv, providers.NewSoftware77(httpClient, v.GetUpdateEvery(), baseDir))
		default:
			return nil, fmt.Errorf("unsupported provider name: %s", v.GetName())
		}
	}

	return rv, nil
}

func makeNewHTTPClient(conf configProvider) topolib.HTTPClient {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	httpClient := &http.Client{
		Timeout: conf.GetHTTPTimeout(),
		Jar:     jar,
	}

	return topolib.NewHTTPClient(httpClient,
		"topographer/"+version,
		conf.GetRateLimitInterval(),
		conf.GetRateLimitBurst())
}

func boolParam(param string) bool {
	switch strings.ToLower(param) {
	case "1", "true", "enabled", "yes":
		return true
	default:
		return false
	}
}

func ensureDir(conf *config, v configProvider) (string, error) {
	baseDir := filepath.Join(conf.GetRootDirectory(), v.GetDirectory())
	if err := os.MkdirAll(baseDir, 0777); err != nil {
		return "", fmt.Errorf("cannot create base directory for dbip provider: %w", err)
	}

	return baseDir, nil
}
