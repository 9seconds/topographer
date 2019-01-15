package providers

import (
	"archive/zip"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	sypex "gopkg.in/night-codes/go-sypexgeo.v1"

	"github.com/9seconds/topographer/config"
	"github.com/juju/errors"
)

const sypexDBURL = "http://sypexgeo.net/files/SxGeoCity_utf8.zip"

// Sypex is a structure for Sypex geolocation provider resolving.
type Sypex struct {
	Provider

	db sypex.SxGEO
}

// Update updates database.
func (sx *Sypex) Update() (bool, error) {
	rawFile, err := sx.downloadURL(sypexDBURL)
	if err != nil {
		return false, errors.Annotate(err, "Cannot update Sypex DB")
	}
	if rawFile == nil {
		return false, errors.Annotate(err, "Cannot update Sypex DB")
	}
	defer func() {
		rawFile.Close()           // nolint
		os.Remove(rawFile.Name()) // nolint
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
		if strings.ToLower(extension) == ".dat" {
			opened, err := zfile.Open()
			if err != nil {
				return false, errors.Annotate(err, "Cannot extract file from archive")
			}
			return sx.saveFile(opened)
		}
	}

	return false, errors.Errorf("Cannot find required file")
}

// Reopen reopens Sypex database.
func (sx *Sypex) Reopen(lastUpdated time.Time) (err error) {
	return sx.reopenSafe(lastUpdated, func() (err error) {
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

		sx.db = sypex.New(sx.FilePath())
		return
	})
}

// Resolve resolves a list of the given IPs
func (sx *Sypex) Resolve(ips []net.IP) ResolveResult {
	return sx.resolveSafe(func() map[string]GeoResult {
		results := make(map[string]GeoResult)

		for _, ip := range ips {
			result := GeoResult{}
			if info, err := sx.db.GetCityFull(ip.String()); err != nil {
				log.WithFields(log.Fields{
					"ip":    ip.String(),
					"error": err.Error(),
				}).Debug("Cannot resolve ip.")
			} else {
				result.Country = sx.extractCountry(info)
				if sx.precision == config.PrecisionCity {
					result.City = sx.extractCity(info)
				}
			}
			results[ip.String()] = result
		}

		return results
	})
}

func (sx *Sypex) extractCountry(info map[string]interface{}) string {
	if countryData, ok := info["country"]; ok {
		if countryMap, ok := countryData.(map[string]interface{}); ok {
			if isoCode, ok := countryMap["iso"]; ok {
				if isoCodeString, ok := isoCode.(string); ok {
					return strings.ToLower(isoCodeString)
				}
			}
		}
	}

	return ""
}

func (sx *Sypex) extractCity(info map[string]interface{}) string {
	if cityData, ok := info["city"]; ok {
		if cityMap, ok := cityData.(map[string]interface{}); ok {
			if cityName, ok := cityMap["name_en"]; ok {
				if cityNameString, ok := cityName.(string); ok {
					return cityNameString
				}
			}
		}
	}

	return ""
}

// NewSypex returns new Sypex geolocation provider structure.
func NewSypex(conf *config.Config) *Sypex {
	return &Sypex{
		Provider: Provider{
			directory:       conf.Directory,
			dbname:          "sypex",
			downloadTimeout: 2 * time.Minute,
			precision:       conf.Precision,
			updateLock:      &sync.RWMutex{},
		},
	}
}
