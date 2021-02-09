package providers

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/9seconds/topographer/topolib"
)

var (
	software77Md5ChecksumRegexp = regexp.MustCompile(`(?i)[0-9a-f]{32}`)
)

const (
	NameSoftware77 = "software77"

	software77IPv4FileName      = "ipv4.csv"
	software77IPv4DownloadParam = "1"
	software77IPv4MD5Param      = "3"

	software77IPv6FileName      = "ipv6.csv"
	software77IPv6DownloadParam = "9"
	software77IPv6MD5Param      = "10"
)

type software77Provider struct {
	db            *software77DB
	dbMutex       sync.RWMutex
	baseDirectory string
	updateEvery   time.Duration
	httpClient    topolib.HTTPClient
}

func (s *software77Provider) Name() string {
	return NameSoftware77
}

func (s *software77Provider) UpdateEvery() time.Duration {
	return s.updateEvery
}

func (s *software77Provider) BaseDirectory() string {
	return s.baseDirectory
}

func (s *software77Provider) Lookup(ctx context.Context, ip net.IP) (topolib.ProviderLookupResult, error) {
	result := topolib.ProviderLookupResult{}

	s.dbMutex.RLock()
	defer s.dbMutex.RUnlock()

	if s.db == nil {
		return result, ErrDatabaseIsNotReadyYet
	}

	res, err := s.db.Lookup(ip)
	if err != nil {
		return result, fmt.Errorf("cannot lookup: %w", err)
	}

	result.CountryCode = res

	return result, nil
}

func (s *software77Provider) Open(rootDir string) error {
	db := newSoftware77DB()

	if err := s.openV4(db, rootDir); err != nil {
		return fmt.Errorf("cannot process db with v4 addresses: %w", err)
	}

	if err := s.openV6(db, rootDir); err != nil {
		return fmt.Errorf("cannot process db with v4 addresses: %w", err)
	}

	s.dbMutex.Lock()
	defer s.dbMutex.Unlock()

	s.db = db

	return nil
}

func (s *software77Provider) openV4(db *software77DB, rootDir string) error {
	fp, err := os.Open(filepath.Join(rootDir, software77IPv4FileName))
	if err != nil {
		return fmt.Errorf("cannot open a file: %w", err)
	}

	defer fp.Close()

	csvReader := csv.NewReader(fp)
	csvReader.Comment = '#'
	csvReader.TrimLeadingSpace = true
	csvReader.ReuseRecord = true

	for {
		record, err := csvReader.Read()

		switch err {
		case nil:
			if err := db.AddIPv4Range(record[0], record[1], record[4]); err != nil {
				return fmt.Errorf("cannot parse a line: %w", err)
			}
		case io.EOF:
			return nil
		default:
			return fmt.Errorf("unexpected error: %w", err)
		}
	}
}

func (s *software77Provider) openV6(db *software77DB, rootDir string) error {
	fp, err := os.Open(filepath.Join(rootDir, software77IPv6FileName))
	if err != nil {
		return fmt.Errorf("cannot open a file: %w", err)
	}

	defer fp.Close()

	csvReader := csv.NewReader(fp)
	csvReader.Comment = '#'
	csvReader.TrimLeadingSpace = true
	csvReader.ReuseRecord = true

	for {
		record, err := csvReader.Read()

		switch err {
		case nil:
			if err := db.AddIPv6CIDR(record[0], record[1]); err != nil {
				return fmt.Errorf("cannot parse a line: %w", err)
			}
		case io.EOF:
			return nil
		default:
			return fmt.Errorf("unexpected error: %w", err)
		}
	}
}

func (s *software77Provider) Shutdown() {
	s.dbMutex.Lock()
	defer s.dbMutex.Unlock()

	s.db = nil
}

func (s *software77Provider) Download(ctx context.Context, rootDir string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 2)
	defer close(errChan)

	wg := &sync.WaitGroup{}

	wg.Add(2)

	go s.downloadCsv(ctx,
		filepath.Join(rootDir, software77IPv4FileName),
		software77IPv4DownloadParam,
		software77IPv4MD5Param,
		errChan,
		wg,
		cancel)

	go s.downloadCsv(ctx,
		filepath.Join(rootDir, software77IPv6FileName),
		software77IPv6DownloadParam,
		software77IPv6MD5Param,
		errChan,
		wg,
		cancel)

	wg.Wait()

	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

func (s *software77Provider) downloadCsv(ctx context.Context,
	filename, downloadParam, md5Param string,
	errChan chan<- error,
	wg *sync.WaitGroup,
	ctxCancel context.CancelFunc) {
	defer wg.Done()

	expectedChecksum, err := s.downloadCsvChecksum(ctx, md5Param)
	if err != nil {
		ctxCancel()
		errChan <- fmt.Errorf("cannot download a checksum: %w", err)

		return
	}

	source, err := s.downloadCsvFile(ctx, downloadParam)
	if err != nil {
		ctxCancel()
		errChan <- fmt.Errorf("cannot download a file: %w", err)

		return
	}

	defer flushResponse(source)

	gzipReader, err := gzip.NewReader(source)
	if err != nil {
		ctxCancel()
		errChan <- fmt.Errorf("cannot create a gzip reader: %w", err)

		return
	}

	defer gzipReader.Close()

	target, err := os.Create(filename)
	if err != nil {
		ctxCancel()
		errChan <- fmt.Errorf("cannot create a target filename: %w", err)

		return
	}

	actualChecksum, err := hashedCopyResponse(md5.New, target, gzipReader)
	if err != nil {
		ctxCancel()
		errChan <- fmt.Errorf("cannot create a copy to a target file: %w", err)

		return
	}

	if !strings.EqualFold(expectedChecksum, actualChecksum) {
		ctxCancel()
		errChan <- fmt.Errorf("checksum mismatch. expected=%s, actual=%s",
			expectedChecksum, actualChecksum)
	}
}

func (s *software77Provider) downloadCsvChecksum(ctx context.Context, md5Param string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		s.buildURL(md5Param), nil)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer flushResponse(resp.Body)

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("cannot read from response body: %w", err)
	}

	content = bytes.TrimSpace(content)

	if !software77Md5ChecksumRegexp.Match(content) {
		return "", fmt.Errorf("incorrect checksum: %s", string(content))
	}

	return string(content), nil
}

func (s *software77Provider) downloadCsvFile(ctx context.Context, downloadParam string) (io.ReadCloser, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		s.buildURL(downloadParam), nil)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (s *software77Provider) buildURL(param string) string {
	getQuery := url.Values{}

	getQuery.Set("DL", param)

	u := url.URL{
		Scheme:   "https",
		Host:     "software77.net",
		Path:     "/geo-ip/",
		RawQuery: getQuery.Encode(),
	}

	return u.String()
}

func NewSoftware77(client topolib.HTTPClient,
	updateEvery time.Duration,
	baseDirectory string) topolib.OfflineProvider {
	return &software77Provider{
		baseDirectory: baseDirectory,
		updateEvery:   updateEvery,
		httpClient:    client,
	}
}
