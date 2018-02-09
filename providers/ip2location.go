package providers

import (
	"archive/zip"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	ip2location "github.com/ip2location/ip2location-go"
	log "github.com/sirupsen/logrus"

	"github.com/9seconds/topographer/config"
	"github.com/juju/errors"
)

const (
	ip2locationDBCodeCountry = "DB1LITEBIN"
	ip2locationDBCodeCity    = "DB3LITEBIN"

	ip2locationDBURL = "https://www.ip2location.com/download/"

	ip2locationTokenEnvName = "TOPOGRAPHER_IP2LOCATION_DOWNLOAD_TOKEN"
)

type IP2Location struct {
	Provider

	dbLock *sync.Mutex
}

func (i2l *IP2Location) Update() (bool, error) {
	token, ok := os.LookupEnv(ip2locationTokenEnvName)
	if !ok {
		return false, errors.Errorf("ip2location download token is not set")
	}

	params := url.Values{}
	params.Set("token", token)
	if i2l.precision == config.PrecisionCountry {
		params.Set("file", ip2locationDBCodeCountry)
	} else {
		params.Set("file", ip2locationDBCodeCity)
	}

	rawFile, err := i2l.downloadURL(ip2locationDBURL + "?" + params.Encode())
	if err != nil {
		return false, errors.Annotatef(err, "Cannot update IP2Location DB")
	}
	defer func() {
		rawFile.Close()
		os.Remove(rawFile.Name())
	}()

	rfStat, err := rawFile.Stat()
	if err != nil {
		return false, errors.Annotate(err, "Cannot stat raw file")
	}

	zipReader, err := zip.NewReader(rawFile, rfStat.Size())
	if err != nil {
		return false, errors.Annotate(err, "Cannot open zip archive")
	}

	for _, zfile := range zipReader.File {
		log.WithFields(log.Fields{
			"filename": zfile.Name,
			"is_dir":   zfile.FileInfo().IsDir(),
		}).Debug("Read file.")

		if zfile.FileInfo().IsDir() {
			continue
		}

		extension := filepath.Ext(zfile.Name)
		if strings.ToLower(extension) == ".bin" {
			if opened, err := zfile.Open(); err != nil {
				return false, errors.Annotate(err, "Cannot extract file from archive")
			} else {
				return i2l.saveFile(opened)
			}
		}
	}

	return false, errors.Errorf("Cannot find required file")
}

func (i2l *IP2Location) Reopen(lastUpdated time.Time) (err error) {
	return i2l.reopenSafe(lastUpdated, func() error {
		if i2l.available {
			ip2location.Close()
		}

		ip2location.Open(i2l.FilePath())

		return nil
	})
}

func (i2l *IP2Location) Resolve(ips []net.IP) ResolveResult {
	return i2l.resolveSafe(func() map[string]GeoResult {
		results := make(map[string]GeoResult)
		for _, ip := range ips {
			results[ip.String()] = i2l.resolveIP(ip)
		}
		return results
	})
}

func (i2l *IP2Location) resolveIP(ip net.IP) GeoResult {
	i2l.dbLock.Lock()
	result := ip2location.Get_all(ip.String())
	i2l.dbLock.Unlock()

	country := strings.ToLower(result.Country_short)
	if country == "invalid database file." || country == "-" {
		country = ""
	}
	georesult := GeoResult{Country: country}

	if i2l.precision == config.PrecisionCity && !strings.Contains(
		result.City,
		"This parameter is unavailable") && result.City != "-" {
		georesult.City = result.City
	}

	return georesult
}

func NewIP2Location(conf *config.Config) *IP2Location {
	return &IP2Location{
		Provider: Provider{
			directory:       conf.Directory,
			dbname:          "ip2location",
			downloadTimeout: 2 * time.Minute,
			precision:       conf.Precision,
			updateLock:      &sync.RWMutex{},
		},
		dbLock: &sync.Mutex{},
	}
}
