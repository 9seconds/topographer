package providers

import (
	"bufio"
	"compress/gzip"
	"io"
	"net"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/9seconds/topographer/csvdb"
	"github.com/asergeyev/nradix"
	"github.com/juju/errors"
)

// CSVDBProvider presents a structure for provider with database in
// CSV format.
type CSVDBProvider struct {
	Provider

	db         *nradix.Tree
	makeRecord csvdb.RecordMaker
}

// Reopen reopens database.
func (cdp *CSVDBProvider) Reopen(lastUpdated time.Time) (err error) {
	return cdp.reopenSafe(lastUpdated, func() error {
		cdp.db = nil
		db, err := cdp.createDatabase()
		if err != nil {
			return err
		}
		cdp.db = db

		return nil
	})
}

func (cdp *CSVDBProvider) createDatabase() (*nradix.Tree, error) {
	rawFile, err := os.Open(cdp.FilePath())
	if err != nil {
		return nil, errors.Annotate(err, "Cannot open database file")
	}

	buferedFile := bufio.NewReader(rawFile)
	gzipFile, err := gzip.NewReader(buferedFile)
	if err != nil {
		return nil, errors.Annotate(err, "Incorrect gzip archive")
	}

	reader := csvdb.NewCSVReader(gzipFile, cdp.makeRecord)
	tree := nradix.NewTree(0)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Annotate(err, "Error during parsing CSV")
		}
		if record == nil {
			continue
		}

		geoData := globalCSVDBCache.get(record.Country, record.City)
		if subnets, err := record.GetSubnets(); err != nil {
			log.WithFields(log.Fields{
				"startIP":  record.StartIP,
				"finishIP": record.FinishIP,
				"err":      err,
			}).Warn("Cannot parse ip range")
		} else {
			for _, cidr := range subnets {
				if errAddOrSet := addOrSetCIDR(tree, cidr, geoData); errAddOrSet != nil {
					return nil, errAddOrSet
				}
			}
		}
	}

	return tree, nil
}

func addOrSetCIDR(tree *nradix.Tree, cidr string, geoData *GeoResult) error {
	if errAdd := tree.AddCIDR(cidr, geoData); errAdd != nil {
		if errAdd == nradix.ErrNodeBusy {
			log.Infof("CIDR %s for country %s already exists. Try to set the new value", cidr, geoData.Country)
			if errSet := tree.SetCIDR(cidr, geoData); errSet != nil {
				return errors.Annotate(errSet, "Incorrect IP range")
			}
			return nil
		} else {
			return errors.Annotate(errAdd, "Incorrect IP range")
		}
	}
	return nil
}

// Resolve resolves a list of the given IPs
func (cdp *CSVDBProvider) Resolve(ips []net.IP) ResolveResult {
	return cdp.resolveSafe(func() map[string]GeoResult {
		results := make(map[string]GeoResult)

		for _, ip := range ips {
			stringIP := ip.String()
			result := GeoResult{}
			if data, err := cdp.db.FindCIDR(stringIP + "/32"); err == nil {
				if converted, ok := data.(*GeoResult); ok {
					result = *converted
				}
			}
			results[stringIP] = result
		}

		return results
	})
}
