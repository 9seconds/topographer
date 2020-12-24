package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/signal"
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
		switch v.GetName() {
		case "ip2c":
			rv = append(rv, providers.NewIP2C(makeNewHTTPClient(v)))
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
		conf.GetRateLimitInterval(),
		conf.GetRateLimitBurst())
}
