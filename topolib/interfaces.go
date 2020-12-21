package topolib

import (
	"net"
	"time"

	"github.com/spf13/afero"
)

type Provider interface {
	Name() string
	Lookup(net.IP) (ProviderLookupResult, error)
}

type OfflineProvider interface {
	Provider

	Start()
	Shutdown()
	UpdateEvery() time.Duration
	BaseDirectory() string
	Open(afero.Fs) error
	Download(afero.Fs) error
}

type Logger interface {
	LookupError(name string, err error)
	UpdateInfo(name string, msg string)
	UpdateError(name string, err error)
}
