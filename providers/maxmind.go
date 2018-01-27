package providers

import (
	"archive/tar"
	"compress/gzip"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	maxminddb "github.com/oschwald/geoip2-golang"
	log "github.com/sirupsen/logrus"

	"github.com/9seconds/topographer/config"
	"github.com/juju/errors"
)

const (
	maxMindDBURLCountry = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-Country.tar.gz"
	maxMindDBURLCity    = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.tar.gz"
)

type MaxMind struct {
	Provider

	db *maxminddb.Reader
}

func (mm *MaxMind) Update() (bool, error) {
	url := maxMindDBURLCountry
	if mm.precision == config.PRECISION_CITY {
		url = maxMindDBURLCity
	}

	rawFile, err := mm.downloadURL(url)
	if err != nil {
		return false, errors.Annotatef(err, "Cannot update MaxMind DB")
	}
	defer func() {
		rawFile.Close()
		os.Remove(rawFile.Name())
	}()

	gzipReader, err := gzip.NewReader(rawFile)
	if err != nil {
		return false, errors.Annotatef(err, "Cannot create GZIP reader")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err != nil || header == nil {
			return false, errors.Errorf("Could not find the file")
		}

		log.WithFields(log.Fields{
			"filename": header.Name,
			"is_dir":   header.FileInfo().IsDir(),
		}).Debug("Read file.")

		if header.FileInfo().IsDir() {
			continue
		}

		extension := filepath.Ext(header.Name)
		if strings.ToLower(extension) == ".mmdb" {
			return mm.saveFile(tarReader)
		}
	}
}

func (mm *MaxMind) Reopen(lastUpdated time.Time) (err error) {
	return mm.reopenSafe(lastUpdated, func() error {
		db, err := maxminddb.Open(mm.FilePath())
		if err != nil {
			return errors.Annotate(err, "Cannot open database")
		}

		if mm.db != nil {
			mm.db.Close()
		}
		mm.db = db

		return nil
	})
}

func (mm *MaxMind) Resolve(ips []net.IP) ResolveResult {
	return mm.resolveSafe(func() map[string]GeoResult {
		results := make(map[string]GeoResult)

		for _, ip := range ips {
			switch mm.precision {
			case config.PRECISION_COUNTRY:
				results[ip.String()] = mm.resolveCountryResult(ip)
			case config.PRECISION_CITY:
				results[ip.String()] = mm.resolveCityResult(ip)
			}
		}

		return results
	})
}

func (mm *MaxMind) resolveCountryResult(ip net.IP) GeoResult {
	result := GeoResult{}

	if country, err := mm.db.Country(ip); err != nil {
		log.WithFields(log.Fields{
			"ip":    ip.String(),
			"error": err.Error(),
		}).Debug("Cannot resolve ip.")
	} else {
		result.Country = strings.ToLower(country.Country.IsoCode)
	}

	return result
}

func (mm *MaxMind) resolveCityResult(ip net.IP) GeoResult {
	result := GeoResult{}

	if city, err := mm.db.City(ip); err != nil {
		log.WithFields(log.Fields{
			"ip":    ip.String(),
			"error": err.Error(),
		}).Debug("Cannot resolve ip.")
	} else {
		if cityName, ok := city.City.Names["en"]; ok {
			result.City = cityName
		}
		result.Country = strings.ToLower(city.Country.IsoCode)
	}

	return result
}

func NewMaxMind(conf *config.Config) *MaxMind {
	return &MaxMind{
		Provider: Provider{
			directory:       conf.Directory,
			dbname:          "maxmind",
			downloadTimeout: time.Minute,
			precision:       conf.Precision,
			updateLock:      &sync.RWMutex{},
		},
	}
}
