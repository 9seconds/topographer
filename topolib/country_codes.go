package topolib

import (
	"strings"

	"github.com/pariz/gountries"
)

type CountryCode uint8

func (c CountryCode) String() string {
	return countryCodeMapCC2String[int(c)]
}

func (c CountryCode) Details() gountries.Country {
	return countryCodeQuery.Countries[c.String()]
}

var (
	countryCodeQuery = gountries.New()

	// empty elements are required to state a 'nil', abscence
	countryCodeMapCC2String = []string{""}
	countryCodeMapString2CC = map[string]CountryCode{"": 0}
)

func NormalizeAlpha2Code(alpha2 string) string {
	alpha2 = strings.ToUpper(alpha2)

	// please read comments in downloaded CSV files
	switch alpha2 {
	case "ZZ", "AP", "EU":
		return ""
	case "YU":
		return "CS"
	case "FX":
		return "FR"
	case "UK":
		return "GB"
	default:
		return alpha2
	}
}

func Alpha2ToCountryCode(alpha2 string) CountryCode {
	return countryCodeMapString2CC[NormalizeAlpha2Code(alpha2)]
}

func init() {
	for k := range countryCodeQuery.Countries {
		k = NormalizeAlpha2Code(k)

		if _, ok := countryCodeMapString2CC[k]; k == "" || ok {
			continue
		}

		countryCodeMapCC2String = append(countryCodeMapCC2String, k)
		countryCodeMapString2CC[k] = CountryCode(len(countryCodeMapCC2String) - 1)
	}
}
