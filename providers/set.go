package providers

import (
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/9seconds/topographer/config"
)

type ProviderSet struct {
	Providers map[string]GeoProvider
	Weights   map[string]float64
}

func (ps *ProviderSet) Update(force bool) {
	var wg sync.WaitGroup

	for k, v := range ps.Providers {
		wg.Add(1)

		go func(name string, provider GeoProvider) {
			defer wg.Done()

			log.WithFields(log.Fields{
				"provider": name,
			}).Info("Update provider.")

			ok, err := provider.Update()
			if err != nil {
				log.WithFields(log.Fields{
					"provider": name,
					"error":    err.Error(),
				}).Warn("Cannot update provider")
			}
			if !force || !ok {
				log.WithFields(log.Fields{
					"provider": name,
				}).Info("Nothing to update")
			} else {
				if err = provider.Reopen(time.Now()); err != nil {
					log.WithFields(log.Fields{
						"provider": name,
					}).Error("Cannot reopen provider database!")
				}
			}
		}(k, v)
	}

	wg.Wait()
}

func (ps *ProviderSet) Resolve(ips []net.IP) []ResolveResult {
	results := make([]ResolveResult, 0, len(ps.Providers))
	channel := make(chan ResolveResult, len(ps.Providers))
	var wg sync.WaitGroup

	resultsCount := 0
	for _, v := range ps.Providers {
		if v.IsReady() {
			wg.Add(1)
			resultsCount += 1

			go func(provider GeoProvider) {
				defer wg.Done()

				channel <- provider.Resolve(ips)
			}(v)
		}
	}

	wg.Wait()
	for i := 0; i < resultsCount; i++ {
		results = append(results, <-channel)
	}

	return results
}

func NewProviderSet(conf *config.Config) ProviderSet {
	set := ProviderSet{
		Providers: make(map[string]GeoProvider),
		Weights:   make(map[string]float64),
	}

	for k, v := range conf.Databases {
		if v.Enabled {
			switch k {
			case "maxmind":
				set.Providers["maxmind"] = NewMaxMind(conf)
			case "dbip":
				set.Providers["dbip"] = NewDBIP(conf)
			case "sypex":
				set.Providers["sypex"] = NewSypex(conf)
			case "ip2location":
				set.Providers["ip2location"] = NewIP2Location(conf)
			}
			set.Weights[k] = v.Weight
		}
	}

	return set
}
