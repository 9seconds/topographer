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
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/9seconds/topographer/topolib"
	"github.com/antchfx/htmlquery"
	"github.com/spf13/afero"
)

var (
	errDBIPFoundNothinOnPage = errors.New("could not find anything on a page")

	dbipUrlRegexp          = regexp.MustCompile(`https?:\/\/download\.db-ip\.com\/free\/.*?\.mmdb\.gz`)
	dbipSha1ChecksumRegexp = regexp.MustCompile(`[0-9a-fA-F]{40}`)
)

const NameDBIP = "dbip"

type dbipProvider struct {
	baseDirectory string
	updateEvery   time.Duration
	client        topolib.HTTPClient
}

func (d *dbipProvider) Name() string {
	return NameDBIP
}

func (d *dbipProvider) Lookup(ctx context.Context, ip net.IP) (topolib.ProviderLookupResult, error) {
	return topolib.ProviderLookupResult{}, nil
}

func (d *dbipProvider) Shutdown() {

}

func (d *dbipProvider) UpdateEvery() time.Duration {
	return d.updateEvery
}

func (d *dbipProvider) BaseDirectory() string {
	return d.baseDirectory
}

func (d *dbipProvider) Open(fs afero.Fs) error {
	return nil
}

func (d *dbipProvider) Download(ctx context.Context, fs afero.Afero) error {
	url, sha1sum, err := d.getFileData(ctx)
	if err != nil {
		return fmt.Errorf("cannot parse html page: %w", err)
	}

	if err := d.downloadFile(ctx, fs, url, sha1sum); err != nil {
		return fmt.Errorf("cannot download a file: %w", err)
	}

	return nil
}

func (d *dbipProvider) getFileData(ctx context.Context) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://db-ip.com/db/download/ip-to-city-lite", nil)
	if err != nil {
		return "", "", fmt.Errorf("cannot compose a request: %w", err)
	}

	htmlPageResp, err := d.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("cannot request a download page: %w", err)
	}

	if htmlPageResp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected http response code: %d", htmlPageResp.StatusCode)
	}

	defer func() {
		io.Copy(ioutil.Discard, htmlPageResp.Body)
		htmlPageResp.Body.Close()
	}()

	tree, err := htmlquery.Parse(bufio.NewReader(htmlPageResp.Body))
	if err != nil {
		return "", "", fmt.Errorf("cannot parse html: %w", err)
	}

	for _, cardNode := range htmlquery.Find(tree, `//div[@class="card"]`) {
		for _, urlNode := range htmlquery.Find(cardNode, `//a[contains(@class, "free_download_link") and @href]`) {
			url := htmlquery.SelectAttr(urlNode, "href")
			if !dbipUrlRegexp.MatchString(url) {
				continue
			}

			for _, ddNode := range htmlquery.Find(cardNode, `//dd[@class="small"]`) {
				text := htmlquery.InnerText(ddNode)
				if dbipSha1ChecksumRegexp.MatchString(text) {
					return url, text, nil
				}
			}
		}
	}

	return "", "", errDBIPFoundNothinOnPage
}

func (d *dbipProvider) downloadFile(ctx context.Context, fs afero.Afero, url, sha1sum string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("cannot compose a request: %w", err)
	}

	fileResp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot download a file: %w", err)
	}

	defer func() {
		io.Copy(ioutil.Discard, fileResp.Body)
		fileResp.Body.Close()
	}()

	fileReader, err := gzip.NewReader(bufio.NewReader(fileResp.Body))
	if err != nil {
		return fmt.Errorf("cannot create a gzip reader: %w", err)
	}

	db, err := fs.Create("database")
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

func NewDBIP(client topolib.HTTPClient, updateEvery time.Duration, baseDirectory string) topolib.OfflineProvider {
	return &dbipProvider{
		client:        client,
		updateEvery:   updateEvery,
		baseDirectory: baseDirectory,
	}
}
