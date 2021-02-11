package topolib

import (
	"bytes"
	"strings"

	"github.com/pariz/gountries"
)

var (
	countryCodeQuery = gountries.New()

	// empty elements are required to state a 'nil', abscence
	countryCodeMapCC2String = []string{""}
	countryCodeMapString2CC = map[string]CountryCode{"": 0}
)

// CountryCode represents a code of the country. Actually it is possible
// to use strings but this custom type is more convenient because it is
// jsonable, stringified and you can query to get more detailed data if
// necessary.
type CountryCode uint8

// MarshalJSON is to conform json.Marshaller interface.
func (c CountryCode) MarshalJSON() ([]byte, error) {
	buf := bytes.Buffer{}

	buf.WriteByte('"')
	buf.WriteString(c.String())
	buf.WriteByte('"')

	return buf.Bytes(), nil
}

// String returns 2-letter ISO3166 country code. For example, for USA it
// is going to be US. For UK - GB.
func (c CountryCode) String() string {
	return countryCodeMapCC2String[int(c)]
}

// Checks if country code is known. You can think about 'known' as
// 'empty' value. Actually we can name it as empty but it is hard for me
// to remember that notion. Known is more natural.
func (c CountryCode) Known() bool {
	return c > 0
}

// Details returns a details for the country taken from gountries.
func (c CountryCode) Details() gountries.Country {
	return countryCodeQuery.Countries[c.String()]
}

// NormalizeAlpha2Code returns a normalized 2-letter ISO3166 code.
// Normalized code is uppercased with some additional mapping. For
// example, some databases return ZZ as 'unknown' country. This function
// returns "" instead. Some databases still map Serbia to YU. This
// correctly maps YU to CS.
//
// So, whenever you want to use 2-letter ISO3166 code and it is coming
// from unknown source, it is recommended to normalize it with this
// function.
func NormalizeAlpha2Code(alpha2 string) string {
	alpha2 = strings.ToUpper(alpha2)

    if len(alpha2) != 2 {
        return ""
    }

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

// Alpha2ToCountryCode maps 2-letter string of ISO3166 to CountryCode
// type.
func Alpha2ToCountryCode(alpha2 string) CountryCode {
	return countryCodeMapString2CC[NormalizeAlpha2Code(alpha2)]
}

// Alpha3ToCountryCode maps 3-letter string of ISO3166 to CountryCode
// type.
func Alpha3ToCountryCode(alpha3 string) CountryCode {
	alpha3 = strings.ToUpper(alpha3)

	return Alpha2ToCountryCode(countryCodeQuery.Alpha3ToAlpha2[alpha3])
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
