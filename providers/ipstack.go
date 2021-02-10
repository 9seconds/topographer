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

const NameIPStack = "ipstack"

type ipstackResponse struct {
	Error struct {
		Code int    `json:"code"`
		Type string `json:"type"`
		Info string `json:"info"`
	} `json:"error"`
	City    string `json:"city"`
	Country string `json:"country_code"`
}

type ipstackProvider struct {
	client     topolib.HTTPClient
	httpScheme string
	authToken  string
}

func (i ipstackProvider) Name() string {
	return NameIPStack
}

func (i ipstackProvider) Lookup(ctx context.Context, ip net.IP) (topolib.ProviderLookupResult, error) {
	result := topolib.ProviderLookupResult{}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, i.buildURL(ip), nil)

	req.Header.Set("Accept", "application/json")

	resp, err := i.client.Do(req)
	if err != nil {
		return result, fmt.Errorf("cannot send a request: %w", err)
	}

    defer flushResponse(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	jsonResponse := ipstackResponse{}
	jsonDecoder := json.NewDecoder(bufio.NewReader(resp.Body))

	if err := jsonDecoder.Decode(&jsonResponse); err != nil {
		return result, fmt.Errorf("cannot parse a response: %w", err)
	}

	if jsonResponse.Error.Code != 0 {
		return result, fmt.Errorf(
			"failed response: code=%d, type=%s, info=%s",
			jsonResponse.Error.Code,
			jsonResponse.Error.Type,
			jsonResponse.Error.Info)
	}

	result.City = jsonResponse.City
	result.CountryCode = topolib.Alpha2ToCountryCode(jsonResponse.Country)

	return result, nil
}

func (i ipstackProvider) buildURL(ip net.IP) string {
	getQuery := url.Values{}

	getQuery.Set("access_key", i.authToken)
	getQuery.Set("output", "json")
	getQuery.Set("fields", "country_code,city")
	getQuery.Set("language", "en")
	getQuery.Set("hostname", "0")
	getQuery.Set("security", "0")

	u := url.URL{
		Scheme:   i.httpScheme,
		Host:     "api.ipstack.com",
		Path:     ip.String(),
		RawQuery: getQuery.Encode(),
	}

	return u.String()
}

func NewIPStack(client topolib.HTTPClient, authToken string, isSecure bool) (topolib.Provider, error) {
	scheme := "http"

	if isSecure {
		scheme = "https"
	}

	if authToken == "" {
		return nil, ErrAuthTokenIsRequired
	}

	return ipstackProvider{
		client:     client,
		authToken:  authToken,
		httpScheme: scheme,
	}, nil
}
