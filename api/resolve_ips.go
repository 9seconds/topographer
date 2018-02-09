package api

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/9seconds/topographer/providers"
)

func resolveIPs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	set := ctx.Value(contextKey("providers")).(*providers.ProviderSet)

	requestBody := ipResolveRequestStruct{}
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		abort(w, http.StatusNotAcceptable, err.Error())
		return
	}
	if len(requestBody.Ips) == 0 {
		abort(w, http.StatusNotAcceptable, "Please provider ips to resolve")
		return
	}

	response := ipResolveResponseStruct{}
	results := set.Resolve(requestBody.Ips, requestBody.Providers)
	response.Build(results)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("Cannot write response: %s", err.Error())
	}
}
