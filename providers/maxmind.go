package providers

import (
	"archive/tar"
	"compress/gzip"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	maxminddb "github.com/oschwald/geoip2-golang"
	log "github.com/sirupsen/logrus"

	"github.com/9seconds/topographer/config"
	"github.com/juju/errors"
)

const (
	maxMindDBURL   = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-Country.tar.gz"
	maxMindTimeout = time.Minute
	maxMindDBName  = "maxmind"
)

type MaxMind struct {
	Provider

	db *maxminddb.Reader
}

func (mm *MaxMind) Update() (bool, error) {
	rawFile, err := mm.DownloadURL(maxMindDBURL, maxMindTimeout)
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

		baseName := filepath.Base(header.Name)
		extension := filepath.Ext(baseName)
		if strings.ToLower(extension) == ".mmdb" {
			return mm.Save(maxMindDBName, tarReader)
		}
	}
}

func (mm *MaxMind) Reopen(lastUpdated time.Time) error {
	db, err := maxminddb.Open(filepath.Join(mm.Directory, maxMindDBName))
	if err != nil {
		return errors.Annotate(err, "Cannot open database")
	}

	mm.Ready = false
	mm.db = db
	mm.LastUpdated = lastUpdated
	mm.Ready = true

	return nil
}

func (mm *MaxMind) Resolve(ips []net.IP) ResolveResult {
	results := ResolveResult{
		ProviderName: maxMindDBName,
		Results:      make(map[string]GeoResult),
	}

	for _, ip := range ips {
		result := GeoResult{}
		if city, err := mm.db.City(ip); err != nil {
			log.WithFields(log.Fields{
				"ip": ip.String(),
			}).Debug("Cannot resolve ip.")
		} else {
			if cityName, ok := city.City.Names["en"]; ok {
				result.City = cityName
			}
			result.Country = strings.ToLower(city.Country.IsoCode)
		}
		results.Results[ip.String()] = result
	}

	return results
}

func NewMaxMind(conf *config.Config) *MaxMind {
	return &MaxMind{Provider: Provider{Directory: conf.Directory}}
}
