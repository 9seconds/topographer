package topolib

import (
	"net"
	"net/http"
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

	h.handleGetIP(w, req, ipAddr)
}

func (h httpHandler) handleGetIP(w http.ResponseWriter, req *http.Request, ipAddr net.IP) {
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
	response := struct {
		Results []*UsageStats `json:"results"`
	}{
		Results: h.topo.UsageStats(),
	}

	h.encodeJSON(w, response)
}
