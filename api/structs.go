package api

import (
	"encoding/json"
	"net"

	"github.com/9seconds/topographer/providers"
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
	Results map[string]*ipResolveItemStruct `json:"results"`
}

func (ir *ipResolveResponseStruct) Build(resolveResults []providers.ResolveResult) {
	ir.Results = make(map[string]*ipResolveItemStruct)
	weights := make(map[string]float64)

	for _, rres := range resolveResults {
		weights[rres.Provider] = rres.Weight

		for ip, data := range rres.Results {
			if _, ok := ir.Results[ip]; !ok {
				ir.Results[ip] = &ipResolveItemStruct{
					Details: make(map[string]ipResolveDetailsItemStruct),
				}
			}

			ir.Results[ip].Details[rres.Provider] = ipResolveDetailsItemStruct{
				Country: data.Country,
				City:    data.City,
			}
		}
	}

	for _, item := range ir.Results {
		country, city := ir.calculateVerdict(weights, item.Details)
		item.City = city
		item.Country = country
	}
}

func (ir *ipResolveResponseStruct) calculateVerdict(
	weights map[string]float64,
	data map[string]ipResolveDetailsItemStruct) (string, string) {
	countryScores := make(map[string]float64)

	for name, details := range data {
		if details.Country != "" {
			if currentValue, ok := countryScores[details.Country]; !ok {
				countryScores[details.Country] = weights[name]
			} else {
				countryScores[details.Country] = currentValue + weights[name]
			}
		}
	}

	country := ir.getWinner(countryScores)
	cityScores := make(map[string]float64)
	for _, details := range data {
		if details.Country == country && details.City != "" {
			if currentValue, ok := cityScores[details.City]; !ok {
				cityScores[details.City] = 1.0
			} else {
				cityScores[details.City] = currentValue + 1.0
			}
		}
	}

	city := ir.getWinner(cityScores)

	return country, city
}

func (ir *ipResolveResponseStruct) getWinner(scores map[string]float64) string {
	winner := ""
	currentMax := 0.0

	for candidate, score := range scores {
		if score >= currentMax {
			winner = candidate
			currentMax = score
		}
	}

	return winner
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
