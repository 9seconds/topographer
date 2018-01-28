package api

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/9seconds/topographer/providers"
)

func selfResolveIP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	set := ctx.Value("providers").(*providers.ProviderSet)

	response := ipResolveResponseStruct{
		Results: make(map[string]*ipResolveItemStruct),
	}
	addr := net.ParseIP(r.RemoteAddr)
	if addr.To4() != nil {
		results := set.Resolve([]net.IP{net.ParseIP(r.RemoteAddr)}, []string{})
		response.Build(results)
	}
	json.NewEncoder(w).Encode(response)

}
