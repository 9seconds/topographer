package providers

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/9seconds/topographer/topolib"
)

type ip2cProvider struct {
	client topolib.HTTPClient
}

func (i ip2cProvider) Name() string {
	return "ip2c"
}

func (i ip2cProvider) Lookup(ctx context.Context, ip net.IP) (topolib.ProviderLookupResult, error) {
	result := topolib.ProviderLookupResult{}
	ip4 := ip.To4()

	if ip4 == nil {
		return result, fmt.Errorf("incorrect ipv4 %v", ip)
	}

	number := strconv.Itoa(int(binary.LittleEndian.Uint32(ip4)))

	req, err := http.NewRequestWithContext(ctx, "GET", "https://ip2c.org/?dec="+number, nil)
	if err != nil {
		return result, fmt.Errorf("cannot build a request: %w", err)
	}

	resp, err := i.client.Do(req)
	defer func() {
		if resp != nil {
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	if err != nil {
		return result, fmt.Errorf("cannot send a request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

    bodyBytes, err := ioutil.ReadAll(bufio.NewReader(resp.Body))
    if err != nil {
        return  result, fmt.Errorf("cannot read response body: %w", err)
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

func NewIP2C(client topolib.HTTPClient) topolib.Provider {
    return ip2cProvider{
        client: client,
    }
}
