package api

import (
	"encoding/json"
	"net"

	"github.com/juju/errors"
)

type providerInfoResponseStruct struct {
	Results map[string]providerInfoItemStruct `json:"results"`
}

type providerInfoItemStruct struct {
	Available   bool    `json:"available"`
	Weight      float64 `json:"weight"`
	LastUpdated int64   `json:"last_updated"`
}

type ipResolveResponseStruct struct {
	Results map[string]ipResolveItemStruct `json:"results"`
}

type ipResolveDetailsItemStruct struct {
	Country string `json:"country"`
	City    string `json:"city"`
}

type ipResolveItemStruct struct {
	ipResolveDetailsItemStruct

	Details map[string]ipResolveDetailsItemStruct `json:"details"`
}

type ipResolveRequestStruct struct {
	Ips       []net.IP
	Providers []string
}

func (req *ipResolveRequestStruct) UnmarshalJSON(text []byte) error {
	raw := struct {
		Ips       []string `json:"ips"`
		Providers []string `json:"providers,omitempty"`
	}{}
	err := json.Unmarshal(text, &raw)
	if err != nil {
		return err
	}

	req.Providers = raw.Providers
	req.Ips = make([]net.IP, 0, len(raw.Ips))
	for _, v := range raw.Ips {
		parsed := net.ParseIP(v)
		if parsed == nil {
			return errors.Errorf("Cannot parse %s as IP", v)
		}
		if parsed.To4() == nil {
			return errors.Errorf("We support only IPv4 (%s)", v)
		}
		req.Ips = append(req.Ips, parsed)
	}

	return nil
}
