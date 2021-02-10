package topolib

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pariz/gountries"
)

var (
	countryCodeQuery = gountries.New()

	// empty elements are required to state a 'nil', abscence
	countryCodeMapCC2String = []string{""}
	countryCodeMapString2CC = map[string]CountryCode{"": 0}
)

type CountryCode uint8

func (c CountryCode) MarshalJSON() ([]byte, error) {
	buf := bytes.Buffer{}

	buf.WriteByte('"')
	buf.WriteString(c.String())
	buf.WriteByte('"')

	return buf.Bytes(), nil
}

func (c *CountryCode) UnmarshalJSON(data []byte) error {
	if cc, ok := countryCodeMapString2CC[string(data)]; ok {
		*c = cc

		return nil
	}

	return fmt.Errorf("incorrect country code %v", data)
}

func (c CountryCode) String() string {
	return countryCodeMapCC2String[int(c)]
}

func (c CountryCode) Known() bool {
	return c > 0
}

func (c CountryCode) Details() gountries.Country {
	return countryCodeQuery.Countries[c.String()]
}

func NormalizeAlpha2Code(alpha2 string) string {
	alpha2 = strings.ToUpper(alpha2)

	// please read comments in downloaded CSV files of software77, this
	// also applicable to ip2c.
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
