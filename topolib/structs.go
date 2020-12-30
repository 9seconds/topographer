package topolib

import "net"

type ResolveResult struct {
	IP      net.IP `json:"ip"`
	Country struct {
		Alpha2Code   string `json:"alpha2_code"`
		Alpha3Code   string `json:"alpha3_code"`
		CommonName   string `json:"common_name"`
		OfficialName string `json:"official_name"`
	} `json:"country"`
	City    string                `json:"city"`
	Details []ResolveResultDetail `json:"details"`
}

func (r *ResolveResult) OK() bool {
	return r.Country.Alpha2Code != "" && r.City != ""
}

type ResolveResultDetail struct {
	ProviderName string `json:"provider_name"`
	CountryCode  string `json:"country_code"`
	City         string `json:"city"`
}

type ProviderLookupResult struct {
	CountryCode string
	City        string
}
