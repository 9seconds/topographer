package topolib

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/spf13/afero"
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
	Open(afero.Fs) error
	Download(context.Context, afero.Afero) error
}

type Logger interface {
	LookupError(ip net.IP, name string, err error)
	UpdateInfo(name, msg string)
	UpdateError(name string, err error)
}

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}
