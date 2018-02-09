package providers

import (
	"bufio"
	"compress/gzip"
	"encoding/csv"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	cidrman "github.com/EvilSuperstars/go-cidrman"
	log "github.com/sirupsen/logrus"

	"github.com/9seconds/topographer/config"
	"github.com/PuerkitoBio/goquery"
	"github.com/asergeyev/nradix"
	"github.com/juju/errors"
)

const (
	dbipDBURLCity    = "https://db-ip.com/db/download/city"
	dbipDBURLCountry = "https://db-ip.com/db/download/country"

	dbipIdxStartIP  = 0
	dbipIdxFinishIP = 1
	dbipIdxCountry  = 2
	dbipIdxCity     = 4

	dbipLRUCacheSize = 256
)

// DBIP presents a structure for db-ip.com provider.
type DBIP struct {
	Provider

	db *nradix.Tree
}

// Update updates database.
func (di *DBIP) Update() (bool, error) {
	initialURL := dbipDBURLCountry
	if di.precision == config.PrecisionCity {
		initialURL = dbipDBURLCity
	}

	archiveURL, err := di.updateGetDownloadLink(initialURL)
	if err != nil {
		return false, errors.Annotate(err, "Cannot get download URL")
	}

	rawFile, err := di.downloadURL(archiveURL)
	if err != nil {
		return false, errors.Annotate(err, "Cannot download DBIP")
	}
	defer func() {
		rawFile.Close()           // nolint
		os.Remove(rawFile.Name()) // nolint
	}()

	return di.saveFile(rawFile)
}

func (di *DBIP) updateGetDownloadLink(url string) (string, error) {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return "", errors.Annotate(err, "Cannot fetch DBIP HTML page")
	}

	url, ok := doc.Find("#free_download_link").First().Attr("href")
	if !ok {
		return "", errors.Errorf("Cannot extract download URL")
	}

	return url, nil
}

// Reopen reopens database.
func (di *DBIP) Reopen(lastUpdated time.Time) (err error) {
	return di.reopenSafe(lastUpdated, func() error {
		di.db = nil
		db, err := di.createDatabase()
		if err != nil {
			return err
		}
		di.db = db

		return nil
	})
}

func (di *DBIP) createDatabase() (*nradix.Tree, error) { // nolint: gocyclo
	rawFile, err := os.Open(di.FilePath())
	if err != nil {
		return nil, errors.Annotate(err, "Cannot open database file")
	}

	buferedFile := bufio.NewReader(rawFile)
	gzipFile, err := gzip.NewReader(buferedFile)
	if err != nil {
		return nil, errors.Annotate(err, "Incorrect gzip archive")
	}

	csvReader := csv.NewReader(gzipFile)
	csvReader.ReuseRecord = true
	tree := nradix.NewTree(0)
	cache := newDBIPCache(dbipLRUCacheSize)

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Annotate(err, "Error during parsing CSV")
		}

		startIPStr := record[dbipIdxStartIP]
		finishIPStr := record[dbipIdxFinishIP]
		country := strings.ToLower(record[dbipIdxCountry])
		city := ""
		if di.precision == config.PrecisionCity {
			city = record[dbipIdxCity]
		}

		startIP := net.ParseIP(startIPStr)
		finishIP := net.ParseIP(finishIPStr)
		if country == "zz" || startIP == nil || startIP.To4() == nil || finishIP == nil || finishIP.To4() == nil {
			continue
		}

		geoData := cache.get(country, city)
		subnets, err := di.getSubnets(startIPStr, finishIPStr)
		if err != nil {
			log.WithFields(log.Fields{
				"startIP":  startIPStr,
				"finishIP": finishIPStr,
				"err":      err,
			}).Warn("Cannot parse ip range")
		} else {
			for _, cidr := range subnets {
				if err := tree.AddCIDR(cidr, geoData); err != nil {
					return nil, errors.Annotate(err, "Incorrect IP range")
				}
			}
		}
	}

	return tree, nil
}

func (di *DBIP) getSubnets(start, finish string) (subnets []string, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			switch x := rec.(type) {
			case string:
				err = errors.Annotate(errors.New(x), "Incorrect subnets")
			case error:
				err = errors.Annotate(x, "Incorrect subnets")
			}
		}
	}()

	subnets, err = cidrman.IPRangeToCIDRs(start, finish)
	return
}

// Resolve resolves a list of the given IPs
func (di *DBIP) Resolve(ips []net.IP) ResolveResult {
	return di.resolveSafe(func() map[string]GeoResult {
		results := make(map[string]GeoResult)

		for _, ip := range ips {
			stringIP := ip.String()
			result := GeoResult{}
			if data, err := di.db.FindCIDR(stringIP + "/32"); err == nil {
				if converted, ok := data.(*GeoResult); ok {
					result = *converted
				}
			}
			results[stringIP] = result
		}

		return results
	})
}

// NewDBIP returns new dbip geolocation provider structure.
func NewDBIP(conf *config.Config) *DBIP {
	return &DBIP{
		Provider: Provider{
			directory:       conf.Directory,
			dbname:          "dbip",
			downloadTimeout: 3 * time.Minute,
			precision:       conf.Precision,
			updateLock:      &sync.RWMutex{},
		},
	}
}
