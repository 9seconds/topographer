package topographer

import "net"

type TestProvider struct{}

func (t TestProvider) Name() string {
	return "tp"
}

func (t TestProvider) Lookup(ip net.IP) (LookupResult, error) {
	return LookupResult{
		CountryCode: "RU",
		City:        "Nizhniy Novgorod",
	}, nil
}
