package providers

import (
	"archive/zip"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	sypex "gopkg.in/night-codes/go-sypexgeo.v1"

	"github.com/9seconds/topographer/config"
	"github.com/juju/errors"
)

const (
	sypexDBURL   = "http://sypexgeo.net/files/SxGeoCity_utf8.zip"
	sypexTimeout = 2 * time.Minute
	sypexDBName  = "sypex"
)

type Sypex struct {
	Provider

	db sypex.SxGEO
}

func (sx *Sypex) Update() (bool, error) {
	rawFile, err := sx.DownloadURL(sypexDBURL, sypexTimeout)
	if err != nil {
		return false, errors.Annotate(err, "Cannot update IP2Location DB")
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
		if strings.ToLower(extension) == ".dat" {
			if opened, err := zfile.Open(); err != nil {
				return false, errors.Annotate(err, "Cannot extract file from archive")
			} else {
				return sx.Save(sypexDBName, opened)
			}
		}
	}

	return false, errors.Errorf("Cannot find required file")
}

func (sx *Sypex) Reopen(lastUpdated time.Time) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			switch x := rec.(type) {
			case string:
				err = errors.Annotate(errors.New(x), "Cannot reopen Sypex database")
			case error:
				err = errors.Annotate(x, "Cannot reopen Sypex database")
			}
		}
	}()

	sx.db = sypex.New(filepath.Join(sx.Directory, sypexDBName))
	sx.LastUpdated = lastUpdated

	return
}

func (sx *Sypex) Resolve(ips []net.IP) ResolveResult {
	results := ResolveResult{
		ProviderName: "sypex",
		Results:      make(map[string]GeoResult),
	}

	for _, ip := range ips {
		result := GeoResult{}
		if info, err := sx.db.GetCityFull(ip.String()); err != nil {
			log.WithFields(log.Fields{
				"ip":    ip.String(),
				"error": err.Error(),
			}).Debug("Cannot resolve ip.")
		} else {
			if countryData, ok := info["country"]; ok {
				if countryMap, ok := countryData.(map[string]interface{}); ok {
					if isoCode, ok := countryMap["iso"]; ok {
						if isoCodeString, ok := isoCode.(string); ok {
							result.Country = strings.ToLower(isoCodeString)
						}
					}
				}
			}
			if cityData, ok := info["city"]; ok {
				if cityMap, ok := cityData.(map[string]interface{}); ok {
					if cityName, ok := cityMap["name_en"]; ok {
						if cityNameString, ok := cityName.(string); ok {
							result.City = cityNameString
						}
					}
				}
			}
		}
		results.Results[ip.String()] = result
	}

	return results
}

func NewSypex(conf *config.Config) *Sypex {
	return &Sypex{Provider: Provider{Directory: conf.Directory}}
}
