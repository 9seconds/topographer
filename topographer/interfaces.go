package topographer

import (
	"net"
	"net/http"
	"time"

	"github.com/pariz/gountries"
)

type LookupResult struct {
	CountryCode string
	City        string
}

type Provider interface {
	Name() string
	Lookup(ip net.IP) (LookupResult, error)
}

type OfflineProvider interface {
	Provider

	Start()
	Shutdown()
	Open() error
	UpdateEvery() time.Duration
	Download(workingDirectory string) error
}

type Logger interface {
	LookupError(name string, err error)
	UpdateInfo(name string, msg string)
	UpdateError(name string, err error)
}

func Handler(opts Opts) http.Handler {
	oopts := &opts
	apiInstance := &api{
		countryQuery: gountries.New(),
		ctx:          opts.Context,
		providers:    make(map[string]Provider),
	}

	for _, v := range opts.Providers {
		apiInstance.providers[v.Name()] = v

		if vv, ok := v.(OfflineProvider); ok {
			go vv.Start()
		}
	}

	go func() {
		<-oopts.Context.Done()

		for _, v := range oopts.Providers {
			if vv, ok := v.(OfflineProvider); ok {
				vv.Shutdown()
			}
		}
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/", apiInstance.HandlerSelf)

	return mux
}
