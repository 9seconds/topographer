package topolib

import (
	"net"
	"net/http"
)

type handleGetResponse struct {
	Result ResolveResult `json:"result"`
}

func (h httpHandler) handleGet(w http.ResponseWriter, req *http.Request) {
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

	respEnvelope := handleGetResponse{
		Result: resolved,
	}

	h.encodeJSON(w, respEnvelope)
}
