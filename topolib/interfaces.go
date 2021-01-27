package topolib

import (
	"context"
	"net"
	"net/http"
	"time"
)

type Provider interface {
	Name() string
	Lookup(context.Context, net.IP) (ProviderLookupResult, error)
}

type OfflineProvider interface {
	Provider

	Shutdown()
	UpdateEvery() time.Duration
	BaseDirectory() string
	Open(string) error
	Download(context.Context, string) error
}

type Logger interface {
	LookupError(ip net.IP, name string, err error)
	UpdateInfo(name, msg string)
	UpdateError(name string, err error)
}

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}
