package providers

import (
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/9seconds/topographer/config"
)

const updateAttempts = 3

// ProviderSet is a structure which combines providers and their weights.
type ProviderSet struct {
	Providers map[string]GeoProvider
	Weights   map[string]float64
}

// Update downloads and applies new databases for all providers.
func (ps *ProviderSet) Update(force bool) {
	for k := range ps.Providers {
		go ps.updateProvider(force, k, 0)
	}
}

func (ps *ProviderSet) updateProvider(force bool, name string, attempt int) {
	provider := ps.Providers[name]

	log.WithFields(log.Fields{
		"provider": name,
		"attempt":  attempt,
	}).Info("Update provider.")

	needToUpdate, err := provider.Update()
	if err != nil {
		log.WithFields(log.Fields{
			"provider": name,
			"attempt":  attempt,
			"error":    err.Error(),
		}).Warn("Cannot update provider")
	} else {
		if needToUpdate || force {
			log.WithFields(log.Fields{
				"provider": name,
				"attempt":  attempt,
			}).Info("Reopening database")

			if err = provider.Reopen(time.Now()); err != nil {
				log.WithFields(log.Fields{
					"provider": name,
					"attempt":  attempt,
					"error":    err.Error(),
				}).Error("Cannot reopen provider database!")
			} else {
				log.WithFields(log.Fields{
					"provider": name,
					"attempt":  attempt,
				}).Info("Database refreshed.")
				return
			}
		} else {
			return
		}
	}

	if attempt < updateAttempts {
		time.Sleep(time.Duration(attempt+1) * time.Minute)
		ps.updateProvider(force, name, attempt+1)
	}
}

// Resolve resolves a list of ips with given list of provider subset to use.
// IF list is empty, it means all providers will be used.
func (ps *ProviderSet) Resolve(ips []net.IP, useProviders []string) []ResolveResult {
	var wg sync.WaitGroup
	results := make([]ResolveResult, 0, len(ps.Providers))
	channel := make(chan ResolveResult, len(ps.Providers))
	defer close(channel)

	providerFilter := make(map[string]bool)
	for _, v := range useProviders {
		providerFilter[v] = true
	}
	if len(useProviders) == 0 {
		for k := range ps.Providers {
			providerFilter[k] = true
		}
	}

	resultsCount := 0
	for k, v := range ps.Providers {
		if _, ok := providerFilter[k]; ok {
			wg.Add(1)
			resultsCount++

			go func(provider GeoProvider) {
				defer wg.Done()

				result := provider.Resolve(ips)
				result.Weight = ps.Weights[result.Provider]
				channel <- result
			}(v)
		}
	}

	wg.Wait()
	for i := 0; i < resultsCount; i++ {
		results = append(results, <-channel)
	}

	return results
}

// NewProviderSet returns a new ProviderSet.
func NewProviderSet(conf *config.Config) *ProviderSet {
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

	return &set
}
