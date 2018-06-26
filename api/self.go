package api

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	"github.com/9seconds/topographer/providers"
)

func selfResolveIP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	set := ctx.Value(contextKey("providers")).(*providers.ProviderSet)

	response := ipResolveResponseStruct{
		Results: make(map[string]*ipResolveItemStruct),
	}

	var ipToResolve net.IP
	givenIP := chi.URLParam(r, "ip")

	if givenIP == "" {
		ipToResolve = net.ParseIP(r.RemoteAddr)
	} else {
		ipToResolve = net.ParseIP(givenIP)
	}

	if ipToResolve.To4() != nil {
		results := set.Resolve([]net.IP{ipToResolve}, []string{})
		response.Build(results)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("Cannot write response: %s", err.Error())
	}
}
