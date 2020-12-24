package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/9seconds/topographer/topolib"
)

const NameKeyCDN = "keycdn"

type keycdnResponse struct {
	Status string `json:"success"`
	Data   struct {
		Geo struct {
			Country string `json:"country_code"`
			City    string `json:"city"`
		} `json:"geo"`
	} `json:"data"`
}

type keycdnProvider struct {
	client topolib.HTTPClient
}

func (k keycdnProvider) Name() string {
	return NameKeyCDN
}

func (k keycdnProvider) Lookup(ctx context.Context, ip net.IP) (topolib.ProviderLookupResult, error) {
	result := topolib.ProviderLookupResult{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://tools.keycdn.com/geo.json?host="+ip.String(), nil)
	if err != nil {
		return result, fmt.Errorf("cannot build a request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := k.client.Do(req)
	if err != nil {
		return result, fmt.Errorf("cannot send a request: %w", err)
	}

	defer func() {
        io.Copy(ioutil.Discard, resp.Body) // nolint: errcheck
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	jsonResponse := keycdnResponse{}
	jsonDecoder := json.NewDecoder(bufio.NewReader(resp.Body))

	if err := jsonDecoder.Decode(&jsonResponse); err != nil {
		return result, fmt.Errorf("cannot parse a response: %w", err)
	}

	if jsonResponse.Status != "success" {
		return result, fmt.Errorf("failed to geolocate: %s", jsonResponse.Status)
	}

	result.City = jsonResponse.Data.Geo.City
	result.CountryCode = jsonResponse.Data.Geo.Country

	return result, nil
}

func NewKeyCDN(client topolib.HTTPClient) topolib.Provider {
	return keycdnProvider{
		client: client,
	}
}
