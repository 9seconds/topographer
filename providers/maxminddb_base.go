package providers

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/9seconds/topographer/topolib"
	"github.com/oschwald/maxminddb-golang"
	"github.com/spf13/afero"
)

type maxmindLookupResult struct {
	City struct {
		Names struct {
			En string `maxminddb:"en"`
		} `maxminddb:"names"`
	} `maxminddb:"city"`
	Country struct {
		IsoCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

type maxmindBase struct {
	dbReader     *maxminddb.Reader
	dbReaderLock sync.RWMutex
}

func (m *maxmindBase) Shutdown() {
	m.dbReaderLock.Lock()
	defer m.dbReaderLock.Unlock()

	if m.dbReader != nil {
		m.dbReader.Close()
		m.dbReader = nil
	}
}

func (m *maxmindBase) Open(fs *afero.BasePathFs) error {
	m.dbReaderLock.Lock()
	defer m.dbReaderLock.Unlock()

	filepath, err := fs.RealPath(dbipLiteFileName)
	if err != nil {
		return fmt.Errorf("cannot resolve a file name of the database: %w", err)
	}

	reader, err := maxminddb.Open(filepath)
	if err != nil {
		return fmt.Errorf("cannot initialize a reader of maxminddb: %w", err)
	}

	if m.dbReader != nil {
		m.dbReader.Close()
	}

	m.dbReader = reader

	return nil
}

func (m *maxmindBase) Lookup(ctx context.Context, ip net.IP) (topolib.ProviderLookupResult, error) {
	m.dbReaderLock.RLock()
	defer m.dbReaderLock.RUnlock()

	rv := topolib.ProviderLookupResult{}

	if m.dbReader == nil {
		return rv, ErrDatabaseIsNotReadyYet
	}

	record := maxmindLookupResult{}

	if err := m.dbReader.Lookup(ip, &record); err != nil {
		return rv, fmt.Errorf("cannot lookup this ip address: %w", err)
	}

	rv.CountryCode = strings.ToUpper(record.Country.IsoCode)
	rv.City = record.City.Names.En

	return rv, nil
}
