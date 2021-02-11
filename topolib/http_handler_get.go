package topolib

import (
	"net"
	"net/http"
	"sort"
)

func (h httpHandler) handleGetResolve(w http.ResponseWriter, req *http.Request) {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		h.sendError(w, err, "Cannot detect your IP address", 0)

		return
	}

	ipAddr := net.ParseIP(host)
	if ipAddr == nil {
		h.sendError(w, nil, "Address was detected incorrectly", 0)

		return
	}

	resolved, err := h.topo.Resolve(req.Context(), ipAddr, nil)
	if err != nil {
		h.sendError(w, err, "Cannot resolve IP address", 0)

		return
	}

	if !resolved.OK() {
		h.sendError(w, nil, "Cannot resolve IP address yet", http.StatusServiceUnavailable)

		return
	}

	response := struct {
		Result ResolveResult `json:"result"`
	}{
		Result: resolved,
	}

	h.encodeJSON(w, response)
}

func (h httpHandler) handleGetStats(w http.ResponseWriter, req *http.Request) {
	stats := make([]*UsageStats, 0, len(h.topo.providerStats))

	for _, v := range h.topo.providerStats {
		stats = append(stats, v)
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Name < stats[j].Name
	})

	response := struct {
		Results []*UsageStats `json:"results"`
	}{
		Results: stats,
	}

	h.encodeJSON(w, response)
}
