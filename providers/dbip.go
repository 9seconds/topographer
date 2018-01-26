package providers

import (
	"bufio"
	"compress/gzip"
	"encoding/csv"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	cidrman "github.com/EvilSuperstars/go-cidrman"
	log "github.com/sirupsen/logrus"

	"github.com/9seconds/topographer/config"
	"github.com/PuerkitoBio/goquery"
	"github.com/asergeyev/nradix"
	"github.com/juju/errors"
)

const (
	dbipDBURL   = "https://db-ip.com/db/download/city"
	dbipTimeout = 3 * time.Minute
	dbipDBName  = "dbip"

	dbipIdxStartIP  = 0
	dbipIdxFinishIP = 1
	dbipIdxCountry  = 2
	dbipIdxCity     = 4
)

var dbipStopIP net.IP = net.ParseIP("255.255.255.255")

type DBIP struct {
	Provider

	db *nradix.Tree
}

func (di *DBIP) Update() (bool, error) {
	archiveUrl, err := di.updateGetDownloadLink()
	if err != nil {
		return false, errors.Annotate(err, "Cannot get download URL")
	}

	rawFile, err := di.DownloadURL(archiveUrl, sypexTimeout)
	if err != nil {
		return false, errors.Annotate(err, "Cannot download DBIP")
	}

	return di.Save(dbipDBName, rawFile)
}

func (di *DBIP) updateGetDownloadLink() (string, error) {
	doc, err := goquery.NewDocument(dbipDBURL)
	if err != nil {
		return "", errors.Annotate(err, "Cannot fetch DBIP HTML page")
	}

	url, ok := doc.Find("#free_download_link").First().Attr("href")
	if !ok {
		return "", errors.Errorf("Cannot extract download URL")
	}

	return url, nil
}

func (di *DBIP) Reopen(lastUpdated time.Time) error {
	di.Ready = false
	di.db = nil
	db, err := di.createDatabase()
	if err != nil {
		return err
	} else {
		di.db = db
	}
	di.LastUpdated = lastUpdated
	di.Ready = true

	return nil
}

func (di *DBIP) Resolve(ips []net.IP) ResolveResult {
	results := ResolveResult{
		ProviderName: "dbip",
		Results:      make(map[string]GeoResult),
	}

	for _, ip := range ips {
		stringIp := ip.String()
		result := GeoResult{}
		if data, err := di.db.FindCIDR(stringIp + "/32"); err == nil {
			result = *(data.(*GeoResult))
		}
		results.Results[stringIp] = result
	}

	return results
}

func (di *DBIP) createDatabase() (*nradix.Tree, error) {
	rawFile, err := os.Open(filepath.Join(di.Directory, dbipDBName))
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
	csvReader.FieldsPerRecord = 5
	tree := nradix.NewTree(0)

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Annotate(err, "Error during parsing CSV")
		}

		startIp := net.ParseIP(record[dbipIdxStartIP])
		finishIp := net.ParseIP(record[dbipIdxFinishIP])
		if startIp == nil || finishIp == nil {
			continue
		}
		if startIp.To4() == nil || finishIp.To4() == nil {
			continue
		}

		country := strings.ToLower(record[dbipIdxCountry])
		if country == "zz" {
			continue
		}
		geoData := &GeoResult{City: record[dbipIdxCity], Country: country}

		subnets, err := di.getSubnets(record[dbipIdxStartIP], record[dbipIdxFinishIP])
		if err != nil {
			log.WithFields(log.Fields{
				"startIp":  record[dbipIdxStartIP],
				"finishIp": record[dbipIdxFinishIP],
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
		if err := recover(); err != nil {
			switch x := err.(type) {
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

func NewDBIP(conf *config.Config) *DBIP {
	return &DBIP{Provider: Provider{Directory: conf.Directory}}
}
