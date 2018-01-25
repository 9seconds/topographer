package providers

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/juju/errors"
)

type Provider struct {
	Ready       bool
	LastUpdated time.Time
	Directory   string
}

type GeoProvider interface {
	Update() (bool, error)
	Resolve(ips []net.IP) []string
}

func (p *Provider) DownloadURL(url string, timeout time.Duration) (*os.File, error) {
	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, errors.Annotate(err, "Cannot create temporary file")
	}

	client := http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, errors.Annotatef(err, "Cannot access URL %s", url)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		io.Copy(ioutil.Discard, resp.Body)

		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, errors.Annotatef(err, "Cannot read from URL %s", url)
	}

	tempFile.Seek(0, 0)

	return tempFile, nil
}

func (p *Provider) Save(filename string, newFile io.Reader) (bool, error) {
	path := filepath.Join(p.Directory, filename)
	fmt.Println(path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			return false, errors.Annotatef(err, "Cannot create file %s", path)
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

	currentFile, err := os.Open(path)
	if err != nil {
		return false, errors.Annotatef(err, "Cannot open file for reading %s", path)
	}

	currentCheckSum := sha1.New()
	io.Copy(currentCheckSum, currentFile)
	currentFile.Close()

	if bytes.Compare(currentCheckSum.Sum(nil), checksum.Sum(nil)) != 0 {
		log.WithFields(log.Fields{
			"current_checksum": hex.EncodeToString(currentCheckSum.Sum(nil)),
			"new_checksum":     hex.EncodeToString(checksum.Sum(nil)),
			"path":             path,
		}).Info("Update database.")

		tempFile.Close()
		if err = os.Rename(path, tempFile.Name()); err != nil {
			return false, errors.Annotatef(err, "Cannot move new file to correct location %s", path)
		}

		return true, nil
	}

	return false, nil
}
