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

// Provider is a structure which represents a basic common data for every
// provider of geolocation information. Basically, all other providers
// should be aggregated with this type.
type Provider struct {
	dbname          string
	directory       string
	updateLock      *sync.RWMutex
	downloadTimeout time.Duration
	lastUpdated     time.Time
	precision       config.Precision
	available       bool
}

// GeoResult represents a basic response on IP geolocation.
type GeoResult struct {
	Country string
	City    string
}

// ResolveResult is a overall result of resolving a list of IPs
// with the given provider sign and its weight.
type ResolveResult struct {
	Provider string
	Weight   float64
	Results  map[string]GeoResult
}

// GeoProvider is the interface which defines a methods each provider
// has to provide or support. Consider it as a public interface.
type GeoProvider interface {
	IsAvailable() bool
	LastUpdated() time.Time
	Reopen(time.Time) error
	Resolve(ips []net.IP) ResolveResult
	Update() (bool, error)
}

func (pr *Provider) reopenSafe(lastUpdated time.Time, callback func() error) (err error) {
	pr.available = false
	pr.updateLock.Lock()
	defer func() {
		pr.updateLock.Unlock()
		if err == nil {
			pr.available = true
		}
	}()

	if err = callback(); err != nil {
		pr.available = false
		return err
	}
	pr.lastUpdated = lastUpdated
	pr.available = true

	return
}

func (pr *Provider) resolveSafe(callback func() map[string]GeoResult) ResolveResult {
	results := make(map[string]GeoResult)

	if pr.available {
		pr.updateLock.RLock()
		defer pr.updateLock.RUnlock()
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

// LastUpdated returns a time when provider updated its database last time
// (reopened with fresh data, not only just downloading).
func (pr *Provider) LastUpdated() time.Time {
	return pr.lastUpdated
}

// IsAvailable tells is provider is available for IP geolocation resolving
// or not.
func (pr *Provider) IsAvailable() bool {
	return pr.available
}

// FilePath returns a path to the database on the disk.
func (pr *Provider) FilePath() string {
	return filepath.Join(pr.directory, pr.dbname)
}

func (pr *Provider) saveFile(newFile io.Reader) (bool, error) {
	if _, err := os.Stat(pr.FilePath()); os.IsNotExist(err) {
		file, err := os.Create(pr.FilePath())
		if err != nil {
			return false, errors.Annotatef(err, "Cannot create file %s", pr.FilePath())
		}
		defer file.Close() // nolint
		if _, err = io.Copy(file, newFile); err != nil {
			return false, errors.Annotate(err, "Cannot copy to the file")
		}

		return true, nil
	}

	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		return false, errors.Annotate(err, "Cannot create temporary file")
	}
	defer func() {
		tempFile.Close()           // nolint
		os.Remove(tempFile.Name()) // nolint
	}()

	checksum := sha1.New()
	writer := io.MultiWriter(checksum, tempFile)
	if _, err = io.Copy(writer, newFile); err != nil {
		return false, errors.Annotate(err, "Cannot copy to the new file")
	}

	currentFile, err := os.Open(pr.FilePath())
	if err != nil {
		return false, errors.Annotatef(err, "Cannot open file for reading %s", pr.FilePath())
	}

	currentCheckSum := sha1.New()
	if _, err = io.Copy(currentCheckSum, currentFile); err != nil {
		return false, errors.Annotate(err, "cannot copy to the new file")
	}
	currentFile.Close() // nolint

	if !bytes.Equal(currentCheckSum.Sum(nil), checksum.Sum(nil)) {
		log.WithFields(log.Fields{
			"current_checksum": hex.EncodeToString(currentCheckSum.Sum(nil)),
			"new_checksum":     hex.EncodeToString(checksum.Sum(nil)),
			"path":             pr.FilePath(),
		}).Info("Update database.")

		tempFile.Close() // nolint
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
		tempFile.Close()           // nolint
		os.Remove(tempFile.Name()) // nolint
		return nil, errors.Annotatef(err, "Cannot access URL %s", url)
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body) // nolint
		resp.Body.Close()                  // nolint
	}()

	log.WithFields(log.Fields{
		"url":       url,
		"code":      resp.StatusCode,
		"body_size": resp.ContentLength,
	}).Debug("Got response.")

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Annotatef(err, "URL gave status code %d", resp.StatusCode)
	}

	if _, err = io.Copy(tempFile, resp.Body); err != nil {
		tempFile.Close()           // nolint
		os.Remove(tempFile.Name()) // nolint
		return nil, errors.Annotatef(err, "Cannot read from URL %s", url)
	}

	log.WithFields(log.Fields{
		"url": url,
	}).Debug("Finish downloading.")

	if _, err = tempFile.Seek(0, 0); err != nil {
		return nil, errors.Annotate(err, "Cannot seek to the start of the file")
	}

	return tempFile, nil
}
