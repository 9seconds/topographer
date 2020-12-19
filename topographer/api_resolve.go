package topographer

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/antzucaro/matchr"
)

type resolveResult struct {
	IP      net.IP `json:"ip"`
	Country struct {
		Alpha2Code   string `json:"alpha2_code"`
		Alpha3Code   string `json:"alpha3_code"`
		CommonName   string `json:"common_name"`
		OfficialName string `json:"official_name"`
	} `json:"country"`
	City    string                `json:"city"`
	Details []resolveResultDetail `json:"details"`
}

type resolveResultDetail struct {
	ProviderName string `json:"provider_name"`
	CountryCode  string `json:"country_code"`
	City         string `json:"city"`
}

func (a *api) Resolve(ips []net.IP, providers []string) ([]resolveResult, error) {
	providersToUse := make([]Provider, 0, len(a.providers))

	if len(providers) == 0 {
		for _, v := range a.providers {
			providersToUse = append(providersToUse, v)
		}
	} else {
		for _, v := range providers {
			vv, ok := a.providers[v]
			if !ok {
				return nil, fmt.Errorf("provider %s is unknown", v)
			}

			providersToUse = append(providersToUse, vv)
		}
	}

	results := make(chan resolveResult, len(ips))
	rv := make([]resolveResult, 0, len(ips))
	wg := &sync.WaitGroup{}

	wg.Add(len(ips))

	go func() {
		wg.Wait()
		close(results)
	}()

	for _, v := range ips {
		go a.resolveIP(v, providersToUse, results, wg)
	}

	for res := range results {
		rv = append(rv, res)
	}

	return rv, nil
}

func (a *api) resolveIP(ip net.IP,
	providers []Provider,
	resultChannel chan<- resolveResult,
	wg *sync.WaitGroup) {
	defer wg.Done()

	rv := make([]resolveResultDetail, 0, len(providers))
	taskChannel := make(chan resolveResultDetail, len(providers))
	wwg := &sync.WaitGroup{}

	wwg.Add(len(providers))

	go func() {
		wwg.Wait()
		close(taskChannel)
	}()

	for _, v := range providers {
		go a.resolveIPLookup(ip, v, taskChannel, wwg)
	}

	for res := range taskChannel {
		rv = append(rv, res)
	}

	select {
	case <-a.ctx.Done():
	case resultChannel <- a.resolveIPMerge(ip, rv):
	}
}

func (a *api) resolveIPLookup(ip net.IP,
	provider Provider,
	taskChannel chan<- resolveResultDetail,
	wg *sync.WaitGroup) {
	defer wg.Done()

	detail := resolveResultDetail{
		ProviderName: provider.Name(),
	}

	res, err := provider.Lookup(ip)
	if err != nil {
		a.logger.LookupError(provider.Name(), err)
	} else {
		detail.City = res.City
		detail.CountryCode = strings.ToUpper(res.CountryCode)
	}

	select {
	case <-a.ctx.Done():
	case taskChannel <- detail:
	}
}

func (a *api) resolveIPMerge(ip net.IP, results []resolveResultDetail) resolveResult {
	countries := map[string][]*resolveResultDetail{}

	for i := range results {
		current := &results[i]

		if current.CountryCode == "" {
			continue
		}

		arr, ok := countries[current.CountryCode]

		if !ok {
			arr = []*resolveResultDetail{}
			countries[current.CountryCode] = arr
		}

		countries[current.CountryCode] = append(arr, current)
	}

	var cityResults []*resolveResultDetail

	maxLen := 0
	selectedCountry := ""

	for country, group := range countries {
		if len(group) > maxLen {
			cityResults = group
			selectedCountry = country
		}
	}

	rv := resolveResult{
		IP:      ip,
		Details: results,
		City:    a.resolveIPMergeCity(cityResults),
	}

	if country, err := a.countryQuery.FindCountryByAlpha(selectedCountry); err == nil {
		rv.Country.Alpha2Code = country.Alpha2
		rv.Country.Alpha3Code = country.Alpha3
		rv.Country.CommonName = country.Name.Common
		rv.Country.OfficialName = country.Name.Official
	}

	return rv

}

func (a *api) resolveIPMergeCity(results []*resolveResultDetail) string {
	counters := make(map[string]int)
	names := make(map[string]string)

	for _, v := range results {
		if v.City == "" {
			continue
		}

		normalizedCityName, _ := matchr.DoubleMetaphone(v.City)

		counters[normalizedCityName] += 1
		names[normalizedCityName] = v.City
	}

	maxLen := 0
	cityName := ""

	for k, v := range counters {
		if v > maxLen {
			cityName = names[k]
			maxLen = v
		}
	}

	return cityName
}
