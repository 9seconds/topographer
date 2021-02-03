package providers

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/9seconds/topographer/topolib"
)

const (
	NameIP2Location = "ip2location"

	ip2locationLiteDB   = "DB3LITEBINIPV6"
	ip2locationFileName = "database.bin"
)

type ip2locationProvider struct {
	dbCode        string
	authToken     string
	baseDirectory string
	updateEvery   time.Duration
	httpClient    topolib.HTTPClient
}

func (i *ip2locationProvider) Name() string {
	return NameIP2Location
}

func (i *ip2locationProvider) UpdateEvery() time.Duration {
	return i.updateEvery
}

func (i *ip2locationProvider) BaseDirectory() string {
	return i.baseDirectory
}

func (i *ip2locationProvider) Lookup(ctx context.Context, ip net.IP) (topolib.ProviderLookupResult, error) {
	result := topolib.ProviderLookupResult{}

	return result, nil
}

func (i *ip2locationProvider) Open(rootDir string) error {
	return nil
}

func (i *ip2locationProvider) Shutdown() {

}

func (i *ip2locationProvider) Download(ctx context.Context, rootDir string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, i.buildURL(), nil)
	if err != nil {
		return fmt.Errorf("cannot build a request: %w", err)
	}

	resp, err := i.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot request a file download: %w", err)
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body) // nolint: errcheck
		resp.Body.Close()
	}()

	tempFile, err := ioutil.TempFile(rootDir, "archive-zip-")
	if err != nil {
		return fmt.Errorf("cannot create a tempfile: %w", err)
	}

	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	if _, err := io.Copy(tempFile, bufio.NewReader(resp.Body)); err != nil {
		return fmt.Errorf("cannot copy archive: %w", err)
	}

	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("cannot seek to the start of the file: %w", err)
	}

	tempFileStat, err := tempFile.Stat()
	if err != nil {
		return fmt.Errorf("cannot stat tempfile: %w", err)
	}

	zipReader, err := zip.NewReader(tempFile, tempFileStat.Size())
	if err != nil {
		return fmt.Errorf("cannot initialize zip reader: %w", err)
	}

	for _, zipFile := range zipReader.File {
		if strings.ToUpper(filepath.Ext(zipFile.Name)) != ".BIN" {
			continue
		}

		dbFile, err := zipFile.Open()
		if err != nil {
			return fmt.Errorf("cannot open a file in archive: %w", err)
		}

		target, err := os.Create(filepath.Join(rootDir, ip2locationFileName))
		if err != nil {
			return fmt.Errorf("cannot create a target file: %w", err)
		}

		if _, err := io.Copy(target, dbFile); err != nil {
			return fmt.Errorf("cannot copy to target file: %w", err)
		}

		return nil
	}

	return fmt.Errorf("cannot find BIN file in archive")
}

func (i *ip2locationProvider) buildURL() string {
	getQuery := url.Values{}

	getQuery.Set("token", i.authToken)
	getQuery.Set("file", i.dbCode)

	u := url.URL{
		Scheme:   "https",
		Host:     "www.ip2location.com",
		Path:     "/download/",
		RawQuery: getQuery.Encode(),
	}

	return u.String()
}

func NewIP2Location(client topolib.HTTPClient,
	updateEvery time.Duration,
	baseDirectory, authToken, dbCode string) (topolib.OfflineProvider, error) {
	if authToken == "" {
		return nil, ErrAuthTokenIsRequired
	}

	if dbCode == "" {
		dbCode = ip2locationLiteDB
	}

	return &ip2locationProvider{
		httpClient:    client,
		updateEvery:   updateEvery,
		baseDirectory: baseDirectory,
		authToken:     authToken,
		dbCode:        dbCode,
	}, nil
}
