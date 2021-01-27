package providers

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/9seconds/topographer/topolib"
	"github.com/antchfx/htmlquery"
)

var (
	dbipLiteErrNothingOnPage   = errors.New("could not find anything on a page")
	dbipLiteUrlRegexp          = regexp.MustCompile(`https?:\/\/download\.db-ip\.com\/free\/.*?\.mmdb\.gz`)
	dbipLiteSha1ChecksumRegexp = regexp.MustCompile(`(?i)[0-9a-f]{40}`)
)

const NameDBIPLite = "dbip_lite"

type dbipLiteProvider struct {
	maxmindBase

	baseDirectory string
	updateEvery   time.Duration
	httpClient    topolib.HTTPClient
}

func (d *dbipLiteProvider) Name() string {
	return NameDBIPLite
}

func (d *dbipLiteProvider) UpdateEvery() time.Duration {
	return d.updateEvery
}

func (d *dbipLiteProvider) BaseDirectory() string {
	return d.baseDirectory
}

func (d *dbipLiteProvider) Download(ctx context.Context, rootDir string) error {
	url, sha1sum, err := d.getFileData(ctx)
	if err != nil {
		return fmt.Errorf("cannot parse html page: %w", err)
	}

	if err := d.downloadFile(ctx, rootDir, url, sha1sum); err != nil {
		return fmt.Errorf("cannot download a file: %w", err)
	}

	return nil
}

func (d *dbipLiteProvider) getFileData(ctx context.Context) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://db-ip.com/db/download/ip-to-city-lite", nil)
	if err != nil {
		return "", "", fmt.Errorf("cannot compose a request: %w", err)
	}

	htmlPageResp, err := d.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("cannot request a download page: %w", err)
	}

	if htmlPageResp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected http response code: %d", htmlPageResp.StatusCode)
	}

	defer func() {
		io.Copy(ioutil.Discard, htmlPageResp.Body) // nolint: errcheck
		htmlPageResp.Body.Close()
	}()

	tree, err := htmlquery.Parse(bufio.NewReader(htmlPageResp.Body))
	if err != nil {
		return "", "", fmt.Errorf("cannot parse html: %w", err)
	}

	for _, cardNode := range htmlquery.Find(tree, `//div[@class="card"]`) {
		for _, urlNode := range htmlquery.Find(cardNode, `//a[contains(@class, "free_download_link") and @href]`) {
			url := htmlquery.SelectAttr(urlNode, "href")
			if !dbipLiteUrlRegexp.MatchString(url) {
				continue
			}

			for _, ddNode := range htmlquery.Find(cardNode, `//dd[@class="small"]`) {
				text := htmlquery.InnerText(ddNode)
				if dbipLiteSha1ChecksumRegexp.MatchString(text) {
					return url, text, nil
				}
			}
		}
	}

	return "", "", dbipLiteErrNothingOnPage
}

func (d *dbipLiteProvider) downloadFile(ctx context.Context, rootDir string, url, sha1sum string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("cannot compose a request: %w", err)
	}

	fileResp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot download a file: %w", err)
	}

	defer func() {
		io.Copy(ioutil.Discard, fileResp.Body) // nolint: errcheck
		fileResp.Body.Close()
	}()

	fileReader, err := gzip.NewReader(fileResp.Body)
	if err != nil {
		return fmt.Errorf("cannot create a gzip reader: %w", err)
	}

	db, err := os.Create(filepath.Join(rootDir, maxmindBaseFileName))
	if err != nil {
		return fmt.Errorf("cannot open a target file: %w", err)
	}

	hasher := sha1.New()

	if _, err := io.Copy(io.MultiWriter(hasher, db), fileReader); err != nil {
		return fmt.Errorf("cannot save a file on filesystem: %w", err)
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))

	if !strings.EqualFold(checksum, sha1sum) {
		return fmt.Errorf("checksum mismatch. expected %s, got %s", sha1sum, checksum)
	}

	return nil
}

func NewDBIPLite(httpClient topolib.HTTPClient, updateEvery time.Duration, baseDirectory string) topolib.OfflineProvider {
	return &dbipLiteProvider{
		httpClient:    httpClient,
		updateEvery:   updateEvery,
		baseDirectory: filepath.Clean(baseDirectory),
	}
}
