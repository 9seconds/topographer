package providers

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/9seconds/topographer/topolib"
)

const NameIP2C = "ip2c"

type ip2cProvider struct {
	client topolib.HTTPClient
}

func (i ip2cProvider) Name() string {
	return NameIP2C
}

func (i ip2cProvider) Lookup(ctx context.Context, ip net.IP) (topolib.ProviderLookupResult, error) {
	result := topolib.ProviderLookupResult{}
	ip4 := ip.To4()

	if ip4 == nil {
		return result, fmt.Errorf("incorrect ipv4 %v", ip)
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		i.buildURL(ip4), nil)

	resp, err := i.client.Do(req)
	if err != nil {
		return result, fmt.Errorf("cannot send a request: %w", err)
	}

	defer flushResponse(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(bufio.NewReader(resp.Body))
	if err != nil {
		return result, fmt.Errorf("cannot read response body: %w", err)
	}

	body := string(bodyBytes)

	chunks := strings.SplitN(body, ";", 3)
	switch {
	case len(chunks) != 3:
		return result, fmt.Errorf("incorrect response: %s", body)
	case chunks[0] != "1":
		return result, fmt.Errorf("ip2c cannot detect region: %s", body)
	}

	result.CountryCode = chunks[1]

	return result, nil
}

func (i ip2cProvider) buildURL(ip net.IP) string {
	getQuery := url.Values{}

	getQuery.Set("dec", strconv.Itoa(int(binary.BigEndian.Uint32(ip))))

	u := url.URL{
		Scheme:   "https",
		Host:     "ip2c.org",
		RawQuery: getQuery.Encode(),
	}

	return u.String()
}

func NewIP2C(client topolib.HTTPClient) topolib.Provider {
	return ip2cProvider{
		client: client,
	}
}
