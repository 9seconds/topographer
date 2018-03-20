package providers

import (
	"os"
	"sync"
	"time"

	"github.com/9seconds/topographer/config"
	"github.com/9seconds/topographer/csvdb"
	"github.com/PuerkitoBio/goquery"
	"github.com/juju/errors"
)

const (
	dbipDBURLCity    = "https://db-ip.com/db/download/city"
	dbipDBURLCountry = "https://db-ip.com/db/download/country"

	dbipIdxStartIP  = 0
	dbipIdxFinishIP = 1
	dbipIdxCountry  = 2
	dbipIdxCity     = 4
)

// DBIP presents a structure for db-ip.com provider.
type DBIP struct {
	CSVDBProvider
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

// NewDBIP returns new dbip geolocation provider structure.
func NewDBIP(conf *config.Config) *DBIP {
	return &DBIP{
		CSVDBProvider: CSVDBProvider{
			Provider: Provider{
				directory:       conf.Directory,
				dbname:          "dbip",
				downloadTimeout: 3 * time.Minute,
				precision:       conf.Precision,
				updateLock:      &sync.RWMutex{},
			},
			makeRecord: func(data []string) (*csvdb.Record, error) {
				city := ""
				if conf.Precision == config.PrecisionCity {
					city = data[dbipIdxCity]
				}

				return csvdb.NewRecord(
					data[dbipIdxCountry],
					city,
					data[dbipIdxStartIP],
					data[dbipIdxFinishIP])
			},
		},
	}
}
