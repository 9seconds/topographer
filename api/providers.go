package api

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/9seconds/topographer/providers"
)

func providerInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	set := ctx.Value("providers").(*providers.ProviderSet)

	response := providerInfoResponseStruct{
		Results: make([]providerInfoItemStruct, 0, len(set.Providers)),
	}
	for name, data := range set.Providers {
		updated := data.LastUpdated().Unix()
		if updated < 0 {
			updated = 0
		}

		item := providerInfoItemStruct{
			Name:        name,
			Available:   data.IsAvailable(),
			Weight:      set.Weights[name],
			LastUpdated: updated,
		}
		response.Results = append(response.Results, item)
	}
	sort.Sort(response)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(response)
}
