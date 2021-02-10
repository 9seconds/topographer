package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

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

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		i.buildURL(ip), nil)

	req.Header.Set("Accept", "application/json")

	if i.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+i.authToken)
	}

	resp, err := i.client.Do(req)
	if err != nil {
		return result, fmt.Errorf("cannot send a request: %w", err)
	}

	defer flushResponse(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	jsonResponse := ipinfoResponse{}
	jsonDecoder := json.NewDecoder(bufio.NewReader(resp.Body))

	if err := jsonDecoder.Decode(&jsonResponse); err != nil {
		return result, fmt.Errorf("cannot parse a response: %w", err)
	}

	result.City = jsonResponse.City
	result.CountryCode = topolib.Alpha2ToCountryCode(jsonResponse.Country)

	return result, nil
}

func (i ipinfoProvider) buildURL(ip net.IP) string {
	u := url.URL{
		Scheme: "https",
		Host:   "ipinfo.io",
		Path:   ip.String(),
	}

	return u.String()
}

func NewIPInfo(client topolib.HTTPClient, authToken string) topolib.Provider {
	return ipinfoProvider{
		authToken: authToken,
		client:    client,
	}
}
