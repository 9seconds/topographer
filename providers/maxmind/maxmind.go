package maxmind

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/9seconds/topographer/config"
	"github.com/9seconds/topographer/providers"
	"github.com/juju/errors"
)

const dbURL = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-Country.tar.gz"
const timeout = time.Minute

type MaxMind struct {
	providers.Provider
}

func (mm *MaxMind) Update() (bool, error) {
	rawFile, err := mm.DownloadURL(dbURL, timeout)
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
			return mm.Save("maxmind", tarReader)
		}
	}
}

func NewProvider(conf *config.Config) *MaxMind {
	return &MaxMind{providers.Provider{Directory: conf.Directory}}
}
