package providers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/9seconds/topographer/topolib"
)

var (
	maxmindChecksumRegexp = regexp.MustCompile(`(?i)[a-f0-9]{64}`)
)

const (
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

func (m *maxmindLiteProvider) Download(ctx context.Context, rootDir string) error {
	expectedChecksum, err := m.downloadChecksum(ctx)
	if err != nil {
		return fmt.Errorf("cannot download a checksum: %w", err)
	}

	actualChecksum, err := m.downloadArchive(ctx, rootDir)
	if err != nil {
		return fmt.Errorf("cannot download an archive")
	}

	if !strings.EqualFold(expectedChecksum, actualChecksum) {
		return fmt.Errorf("checksum mismatch. expected=%s, actual=%s",
			expectedChecksum,
			actualChecksum)
	}

	if err := m.extractArchive(rootDir); err != nil {
		return fmt.Errorf("cannot extract archive: %w", err)
	}

	return nil
}

func (m *maxmindLiteProvider) downloadChecksum(ctx context.Context) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", m.buildURL("tar.gz.sha256"), nil)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot fetch checksum page: %w", err)
	}

	defer flushResponse(resp.Body)

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

func (m *maxmindLiteProvider) downloadArchive(ctx context.Context, rootDir string) (string, error) {
	tarFile, err := os.Create(filepath.Join(rootDir, maxmindLiteArchiveName))
	if err != nil {
		return "", fmt.Errorf("cannot create an archive file: %w", err)
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", m.buildURL("tar.gz"), nil)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot download an archive: %w", err)
	}

	defer flushResponse(resp.Body)

	pipeReadEnd, pipeWriteEnd := io.Pipe()

	defer pipeReadEnd.Close()
	defer pipeWriteEnd.Close()

	errChan := make(chan error)

	go func() {
		_, err := io.Copy(tarFile, pipeReadEnd)
		errChan <- err
	}()

	checksum, err := hashedCopyResponse(sha256.New, pipeWriteEnd, resp.Body)
	if err != nil {
		return "", fmt.Errorf("cannot copy file into fs: %w", err)
	}

	pipeWriteEnd.Close()

	if err := <-errChan; err != nil {
		return "", fmt.Errorf("cannot write to tar file: %w", err)
	}

	return checksum, nil
}

func (m *maxmindLiteProvider) extractArchive(rootDir string) error {
	archiveFile, err := os.Open(filepath.Join(rootDir, maxmindLiteArchiveName))
	if err != nil {
		return fmt.Errorf("cannot open archive: %w", err)
	}

	databaseFile, err := os.Create(filepath.Join(rootDir, maxmindBaseFileName))
	if err != nil {
		return fmt.Errorf("cannot create a file for a database: %w", err)
	}

	defer os.Remove(filepath.Join(rootDir, maxmindLiteArchiveName))

	ungzipReader, err := gzip.NewReader(archiveFile)
	if err != nil {
		return fmt.Errorf("cannot create a gzip reader: %w", err)
	}

	tarReader := tar.NewReader(ungzipReader)

	for {
		header, err := tarReader.Next()

		switch {
		case err == io.EOF:
			return ErrNoFile
		case err != nil:
			return fmt.Errorf("cannot extract a header: %w", err)
		case header.Linkname != "", header.FileInfo().IsDir():
			continue
		case strings.ToUpper(filepath.Ext(header.Name)) == ".MMDB":
			if _, err := io.Copy(databaseFile, tarReader); err != nil {
				return fmt.Errorf("cannot copy into a database file: %w", err)
			}

			return nil
		}
	}
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

// NewMaxmindLite returns a new instance which works with lite
// databases from MaxMind.
//
//   Identifier: maxmind_lite
//   Provider type: offline
//   Website: https://maxmind.com
//
// Probably a main choice if we speak on IP geolocation
// databases. The biggest player in this field.
func NewMaxmindLite(httpClient topolib.HTTPClient,
	updateEvery time.Duration,
	baseDirectory string,
	licenseKey string) (topolib.OfflineProvider, error) {
	if licenseKey == "" {
		return nil, ErrAuthTokenIsRequired
	}

	return &maxmindLiteProvider{
		httpClient:    httpClient,
		updateEvery:   updateEvery,
		baseDirectory: filepath.Clean(baseDirectory),
		licenseKey:    licenseKey,
	}, nil
}
