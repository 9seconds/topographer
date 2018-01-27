package providers

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/9seconds/topographer/config"
	"github.com/juju/errors"
)

type Provider struct {
	available       bool
	directory       string
	dbname          string
	lastUpdated     time.Time
	downloadTimeout time.Duration
	updateLock      *sync.RWMutex
	precision       config.Precision
}

type GeoResult struct {
	Country string
	City    string
}

type ResolveResult struct {
	Provider string
	Weight   float64
	Results  map[string]GeoResult
}

type GeoProvider interface {
	IsAvailable() bool
	LastUpdated() time.Time
	Reopen(time.Time) error
	Resolve(ips []net.IP) ResolveResult
	Update() (bool, error)
}

func (pr *Provider) reopenSafe(lastUpdated time.Time, callback func() error) error {
	pr.updateLock.Lock()
	defer pr.updateLock.Unlock()

	if err := callback(); err != nil {
		pr.available = false
		return err
	}
	pr.lastUpdated = lastUpdated
	pr.available = true

	return nil
}

func (pr *Provider) resolveSafe(callback func() map[string]GeoResult) ResolveResult {
	pr.updateLock.RLock()
	defer pr.updateLock.RUnlock()

	results := make(map[string]GeoResult)
	if pr.available {
		results = callback()
	} else {
		log.WithFields(log.Fields{
			"provider": pr.dbname,
		}).Warn("Provider is not available for resolving.")
	}

	return ResolveResult{
		Provider: pr.dbname,
		Results:  results,
	}
}

func (pr *Provider) LastUpdated() time.Time {
	return pr.lastUpdated
}

func (pr *Provider) IsAvailable() bool {
	return pr.available
}

func (pr *Provider) FilePath() string {
	return filepath.Join(pr.directory, pr.dbname)
}

func (pr *Provider) saveFile(newFile io.Reader) (bool, error) {
	if _, err := os.Stat(pr.FilePath()); os.IsNotExist(err) {
		file, err := os.Create(pr.FilePath())
		if err != nil {
			return false, errors.Annotatef(err, "Cannot create file %s", pr.FilePath())
		}
		defer file.Close()
		io.Copy(file, newFile)

		return true, nil
	}

	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		return false, errors.Annotate(err, "Cannot create temporary file")
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	checksum := sha1.New()
	writer := io.MultiWriter(checksum, tempFile)
	io.Copy(writer, newFile)

	currentFile, err := os.Open(pr.FilePath())
	if err != nil {
		return false, errors.Annotatef(err, "Cannot open file for reading %s", pr.FilePath())
	}

	currentCheckSum := sha1.New()
	io.Copy(currentCheckSum, currentFile)
	currentFile.Close()

	if bytes.Compare(currentCheckSum.Sum(nil), checksum.Sum(nil)) != 0 {
		log.WithFields(log.Fields{
			"current_checksum": hex.EncodeToString(currentCheckSum.Sum(nil)),
			"new_checksum":     hex.EncodeToString(checksum.Sum(nil)),
			"path":             pr.FilePath(),
		}).Info("Update database.")

		tempFile.Close()
		if err = os.Rename(tempFile.Name(), pr.FilePath()); err != nil {
			return false, errors.Annotatef(err, "Cannot move new file to correct location %s", pr.FilePath())
		}

		return true, nil
	}

	return false, nil
}

func (pr *Provider) downloadURL(url string) (*os.File, error) {
	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, errors.Annotate(err, "Cannot create temporary file")
	}

	log.WithFields(log.Fields{
		"url": url,
	}).Debug("Access URL.")

	client := http.Client{Timeout: pr.downloadTimeout}
	resp, err := client.Get(url)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, errors.Annotatef(err, "Cannot access URL %s", url)
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	log.WithFields(log.Fields{
		"url":       url,
		"code":      resp.StatusCode,
		"body_size": resp.ContentLength,
	}).Debug("Got response.")

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Annotatef(err, "URL gave status code %d", resp.StatusCode)
	}

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, errors.Annotatef(err, "Cannot read from URL %s", url)
	}

	tempFile.Seek(0, 0)

	return tempFile, nil
}
