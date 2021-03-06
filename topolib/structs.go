package topolib

import "net"

// ResolveResult is a consolidated verdict on IP geolocation
// made by Topographer.
//
// Consolidation means that there is a process of 'voting' for
// the location. Currently it is done in a following way:
//
// 1. All providers give their results
//
// 2. Topographer choose the most 'popular country'
//
// 3. It choose the most 'popular city' among these from step 2.
//
// 4. This is a verdict.
type ResolveResult struct {
	// IP is IP address which we resolve.
	IP net.IP `json:"ip"`

	// Country is a set of details on a choosen country.
	Country struct {
		// Alpha2Code is 2-letter ISO3166 code. For example, for Russia
		// it is going to be RU.
		Alpha2Code string `json:"alpha2_code"`

		// Alpha3Code is 3-letter ISO3166 code. For example, for Russia
		// it is going to be RUS.
		Alpha3Code string `json:"alpha3_code"`

		// CommonName is a name of the country which we use in real life.
		// Like, Russia.
		CommonName string `json:"common_name"`

		// OfficialName is a name of the country which we use in
		// official papers. Like Russian Federation.
		OfficialName string `json:"official_name"`
	} `json:"country"`

	// City is a name of the city this IP belongs to.
	City string `json:"city"`

	// Details is a list of 'raw' results generated by Providers. If you
	// are interested in why Topographer choose this country or what was
	// the choise of ipinfo.io, this is a field to go.
	Details []ResolveResultDetail `json:"details"`
}

// OK checks if response has some data. For example, it is possible that
// we have no city or country.
func (r *ResolveResult) OK() bool {
	return r.Country.Alpha2Code != "" && r.City != ""
}

// ResolveResultDetail is a result generated by Provider.
type ResolveResultDetail struct {
	// ProviderName is a name of the provider which made
	// this choice.
	ProviderName string `json:"provider_name"`

	// CountryCode is a code of the chosen country.
	CountryCode CountryCode `json:"country_code"`

	// City is a name of the city.
	City string `json:"city"`
}

// ProviderLookupResult is a strucutre which is returned by Provider
// interface. It is not the same as ResolveResultDetail. A latter one is
// used in consolidated responses while ProviderLookupResult should be
// used by those who want to implement their own providers.
type ProviderLookupResult struct {
	// CountryCode is a code of the chosen country.
	CountryCode CountryCode

	// City is the name of the city.
	City string
}
