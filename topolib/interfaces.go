package topolib

import (
	"context"
	"net"
	"net/http"
	"time"
)

// Provider represents an entity which can resolve geolocation of IP
// addresses.
//
// Each provider should work with its own service. For example, if you
// use MaxMind, then there should be a dedicated provider for MaxMind.
type Provider interface {
	// Name returns a unique identifier of the provider.
	// Usually a name.
	Name() string

	// Lookup resolves a location of IP address. If geolocation fails
	// for some reason, it has to return an error. There is no need to
	// fill each field of ProviderLookupResult but it is expected that
	// at least CountryCode is present.
	Lookup(context.Context, net.IP) (ProviderLookupResult, error)
}

// OfflineProvider is a special version of Provider which works with
// offline databases.
//
// A major difference is that these databases should be downloaded
// opened and updated on periodic basis. This all is responsibility
// of Topographer instance. It pass correct directories to Open and
// Download method and ensures they are writable/readable.
type OfflineProvider interface {
	Provider

	// Shutdown shuts down an offline provider. Once it is shutdown,
	// topographer won't call any methods of this provider except of
	// Name.
	Shutdown()

	// UpdateEvery returns a periodicity which should be used to update
	// databases.
	UpdateEvery() time.Duration

	// BaseDirectory return a base directory for this offline provider.
	// A base directory is that one which has target, tmps and so on.
	// Provider should not write anywhere outside of this directory.
	BaseDirectory() string

	// Open ingests a new database update from FS. It is expected
	// that on failures all old handles are alive. There should be no
	// situation that database is updated and this update breaks a
	// database.
	//
	// There is an open question if we have to keep old working db or
	// not. Right now topographer deletes old database so it is possible
	// that random restart of the service will make Provider broken.
	// This is intentional. If you want to do a validation, please do it
	// in Download method.
	Open(string) error

	// Download takes a directory and creates a file structure which is
	// suitable for Open method. Also, it is a place where you can do a
	// validation like checksum match etc. If download returns no error,
	// it means that database is working and should be used.
	Download(context.Context, string) error
}

// Logger is a logger interface used by Topographer.
//
// Each method accepts name parameter. name is a name of the provider.
type Logger interface {
	// LookupError is a method which logs different errors
	// related to IP address lookup.
	LookupError(ip net.IP, name string, err error)

	// UpdateInfo notifies that provider has updated a database with
	// no errors.
	UpdateInfo(name string)

	// UpdateError notifies that there was an error in updated database.
	UpdateError(name string, err error)
}

// HTTPClient is something which can do HTTP requests and returns HTTP
// responses.
//
// For example, topolib enrich default http.Client with rate limiters
// and circuit breakers. If you want, you can build your own client.
//
// It is also important that topographer is designed in a way that each
// provider has to use its own HTTP client. There should be no sharing
// of such clients between different instances.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}
