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
	getQuery := url.Values{}

	getQuery.Set("access_key", i.authToken)
	getQuery.Set("output", "json")
	getQuery.Set("fields", "country_code,city")
	getQuery.Set("language", "en")
	getQuery.Set("hostname", "0")
	getQuery.Set("security", "0")

	u := &url.URL{
		Scheme:   i.httpScheme,
		Host:     "api.ipstack.com",
		Path:     ip.String(),
		RawQuery: getQuery.Encode(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return result, fmt.Errorf("cannot build a request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := i.client.Do(req)
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
	result.CountryCode = jsonResponse.Country

	return result, nil
}

func NewIPStack(client topolib.HTTPClient, authToken string, isSecure bool) topolib.Provider {
	scheme := "http"

	if isSecure {
		scheme = "https"
	}

	return ipstackProvider{
		client:     client,
		authToken:  authToken,
		httpScheme: scheme,
	}
}
