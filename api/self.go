package api

import (
	"encoding/json"
	"net"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/9seconds/topographer/providers"
)

func selfResolveIP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	set := ctx.Value(contextKey("providers")).(*providers.ProviderSet)

	response := ipResolveResponseStruct{
		Results: make(map[string]*ipResolveItemStruct),
	}
	addr := net.ParseIP(r.RemoteAddr)
	if addr.To4() != nil {
		results := set.Resolve([]net.IP{net.ParseIP(r.RemoteAddr)}, []string{})
		response.Build(results)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("Cannot write response: %s", err.Error())
	}
}
