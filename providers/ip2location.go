package providers

import (
	"archive/zip"
	"net"
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
	ip2locationDBURL   = "http://download.ip2location.com/lite/IP2LOCATION-LITE-DB1.BIN.ZIP"
	ip2locationTimeout = time.Minute
	ip2locationDBName  = "ip2location"
)

type IP2Location struct {
	Provider

	dbLock *sync.Mutex
}

func (i2l *IP2Location) Update() (bool, error) {
	rawFile, err := i2l.DownloadURL(ip2locationDBURL, ip2locationTimeout)
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

		baseName := filepath.Base(zfile.Name)
		extension := filepath.Ext(baseName)
		if strings.ToLower(extension) == ".bin" {
			if opened, err := zfile.Open(); err != nil {
				return false, errors.Annotate(err, "Cannot extract file from archive")
			} else {
				return i2l.Save(ip2locationDBName, opened)
			}
		}
	}

	return false, errors.Errorf("Cannot find required file")
}

func (i2l *IP2Location) Reopen(lastUpdated time.Time) error {
	i2l.dbLock.Lock()
	defer i2l.dbLock.Unlock()

	i2l.Ready = false
	if i2l.Ready {
		ip2location.Close()
	}

	ip2location.Open(filepath.Join(i2l.Directory, ip2locationDBName))
	i2l.LastUpdated = lastUpdated
	i2l.Ready = true

	return nil
}

func (i2l *IP2Location) Resolve(ips []net.IP) ResolveResult {
	results := ResolveResult{
		ProviderName: "ip2location",
		Results:      make(map[string]GeoResult),
	}

	for _, ip := range ips {
		results.Results[ip.String()] = i2l.ResolveIP(ip)
	}

	return results
}

func (i2l *IP2Location) ResolveIP(ip net.IP) GeoResult {
	i2l.dbLock.Lock()
	result := ip2location.Get_all(ip.String())
	i2l.dbLock.Unlock()

	georesult := GeoResult{Country: strings.ToLower(result.Country_short)}

	if !strings.Contains(result.City, "This parameter is unavailable") {
		georesult.City = result.City
	}

	return georesult
}

func NewIP2Location(conf *config.Config) *IP2Location {
	return &IP2Location{
		Provider: Provider{Directory: conf.Directory},
		dbLock:   &sync.Mutex{},
	}
}
