package providers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/9seconds/topographer/topolib"
	"github.com/spf13/afero"
)

var (
	maxmindChecksumRegexp = regexp.MustCompile(`(?i)[a-f0-9]{64}`)
)

const (
	NameMaxmindLite = "maxmind_lite"

	maxmindLiteArchiveName = "archive.tar.gz"
)

type maxmindLiteProvider struct {
	maxmindBase

	baseDirectory string
	licenseKey    string
	updateEvery   time.Duration
	httpClient    topolib.HTTPClient
}

func (m *maxmindLiteProvider) Name() string {
	return NameMaxmindLite
}

func (m *maxmindLiteProvider) UpdateEvery() time.Duration {
	return m.updateEvery
}

func (m *maxmindLiteProvider) BaseDirectory() string {
	return m.baseDirectory
}

func (m *maxmindLiteProvider) Download(ctx context.Context, fs afero.Afero) error {
	expectedChecksum, err := m.downloadChecksum(ctx)
	if err != nil {
		return fmt.Errorf("cannot download a checksum: %w", err)
	}

	actualChecksum, err := m.downloadArchive(ctx, fs)
	if err != nil {
		return fmt.Errorf("cannot download an archive")
	}

	if !strings.EqualFold(expectedChecksum, actualChecksum) {
		return fmt.Errorf("checksum mismatch. expected=%s, actual=%s",
			expectedChecksum,
			actualChecksum)
	}

	if err := m.extractArchive(fs); err != nil {
		return fmt.Errorf("cannot extract archive: %w", err)
	}

	return nil
}

func (m *maxmindLiteProvider) downloadChecksum(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.buildURL("tar.gz.sha256"), nil)
	if err != nil {
		panic(err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot fetch checksum page: %w", err)
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("cannot read body of the response: %w", err)
	}

	pos := bytes.IndexAny(data, " \t")
	if pos == -1 {
		return "", fmt.Errorf("incorrect response format: %w", err)
	}

	if !maxmindChecksumRegexp.Match(data[:pos]) {
		return "", fmt.Errorf("incorrect checksum format: %w", err)
	}

	return string(data[:pos]), nil
}

func (m *maxmindLiteProvider) downloadArchive(ctx context.Context, fs afero.Afero) (string, error) {
	tarFile, err := fs.Create(maxmindLiteArchiveName)
	if err != nil {
		return "", fmt.Errorf("cannot create an archive file: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", m.buildURL("tar.gz"), nil)
	if err != nil {
		panic(err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot download an archive: %w", err)
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	pipeReadEnd, pipeWriteEnd := io.Pipe()

	defer pipeReadEnd.Close()
	defer pipeWriteEnd.Close()

	hasher := sha256.New()
	errChan := make(chan error)
	writer := io.MultiWriter(hasher, pipeWriteEnd)

	go func() {
		_, err := io.Copy(tarFile, pipeReadEnd)
		errChan <- err
	}()

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return "", fmt.Errorf("cannot copy file into fs: %w", err)
	}

	pipeWriteEnd.Close()

	if err := <-errChan; err != nil {
		return "", fmt.Errorf("cannot write to tar file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (m *maxmindLiteProvider) extractArchive(fs afero.Afero) error {
	archiveFile, err := fs.Open(maxmindLiteArchiveName)
	if err != nil {
		return fmt.Errorf("cannot open archive: %w", err)
	}

	databaseFile, err := fs.Create(maxmindBaseFileName)
	if err != nil {
		return fmt.Errorf("cannot create a file for a database: %w", err)
	}

	ungzipReader, err := gzip.NewReader(archiveFile)
	if err != nil {
		return fmt.Errorf("cannot create a gzip reader: %w", err)
	}

	tarReader := tar.NewReader(ungzipReader)

	for {
		header, err := tarReader.Next()
		if err != nil {
			return fmt.Errorf("cannot extract a header: %w", err)
		}

		if header.Linkname != "" || header.FileInfo().IsDir() {
			continue
		}

		if filepath.Ext(header.Name) == ".mmdb" {
			break
		}
	}

	if _, err := io.Copy(databaseFile, tarReader); err != nil {
		return fmt.Errorf("cannot copy into a database file: %w", err)
	}

    fs.Remove(maxmindLiteArchiveName)

	return nil
}

func (m *maxmindLiteProvider) buildURL(suffix string) string {
	queryValues := url.Values{}

	queryValues.Set("edition_id", "GeoLite2-City")
	queryValues.Set("suffix", suffix)
	queryValues.Set("license_key", m.licenseKey)

	urlStruct := url.URL{
		Scheme:   "https",
		Host:     "download.maxmind.com",
		Path:     "/app/geoip_download",
		RawQuery: queryValues.Encode(),
	}

	return urlStruct.String()
}

func NewMaxmindLite(httpClient topolib.HTTPClient,
	updateEvery time.Duration,
	baseDirectory string,
	parameters map[string]string) topolib.OfflineProvider {
	return &maxmindLiteProvider{
		httpClient:    httpClient,
		updateEvery:   updateEvery,
		baseDirectory: baseDirectory,
		licenseKey:    parameters["license_key"],
	}
}
