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

const NameIPInfo = "ipinfo"

type ipinfoResponse struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

type ipinfoProvider struct {
	authToken string
	client    topolib.HTTPClient
}

func (i ipinfoProvider) Name() string {
	return NameIPInfo
}

func (i ipinfoProvider) Lookup(ctx context.Context, ip net.IP) (topolib.ProviderLookupResult, error) {
	result := topolib.ProviderLookupResult{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ipinfo.io/"+ip.String(), nil)
	if err != nil {
		return result, fmt.Errorf("cannot build a request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	if i.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+i.authToken)
	}

	resp, err := i.client.Do(req)
	if err != nil {
		return result, fmt.Errorf("cannot send a request: %w", err)
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	jsonResponse := ipinfoResponse{}
	jsonDecoder := json.NewDecoder(bufio.NewReader(resp.Body))

	if err := jsonDecoder.Decode(&jsonResponse); err != nil {
		return result, fmt.Errorf("cannot parse a response: %w", err)
	}

	result.City = jsonResponse.City
	result.CountryCode = jsonResponse.Country

	return result, nil
}

func NewIPInfo(client topolib.HTTPClient, parameters map[string]string) topolib.Provider {
	return ipinfoProvider{
		authToken: parameters["auth_token"],
		client:    client,
	}
}
