package providers

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/9seconds/topographer/topolib"
	"github.com/spf13/afero"
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://db-ip.com/db/download/ip-to-city-lite", nil)
	if err != nil {
		return fmt.Errorf("cannot compose a request: %w", err)
	}

	htmlPageResp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot request a download page: %w", err)
	}

	if htmlPageResp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected http response code: %d", htmlPageResp.StatusCode)
	}

	defer func() {
		io.Copy(ioutil.Discard, htmlPageResp.Body)
		htmlPageResp.Body.Close()
	}()

	fs.WriteReader("page", htmlPageResp.Body)

	return nil
}

func NewDBIP(client topolib.HTTPClient, updateEvery time.Duration, baseDirectory string) topolib.OfflineProvider {
	return &dbipProvider{
		client:        client,
		updateEvery:   updateEvery,
		baseDirectory: baseDirectory,
	}
}
