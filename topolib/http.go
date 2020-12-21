package topolib

import (
	"encoding/json"
	"net"
	"net/http"
)

type httpHandler struct {
	topo *Topographer
}

func (h httpHandler) HandleSelf(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" && req.Method != "HEAD" {
		h.sendError(w, nil, "This HTTP method is not allowed", http.StatusMethodNotAllowed)

		return
	}

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

	respEnvelope := struct {
		Result ResolveResult `json:"result"`
	}{
		Result: resolved,
	}

	h.encodeJSON(w, respEnvelope)
}

func (h httpHandler) encodeJSON(w http.ResponseWriter, data interface{}) {
	encoder := json.NewEncoder(w)

	w.Header().Add("Content-Type", "application/json")
	encoder.SetEscapeHTML(false)
	encoder.Encode(data) // nolint: errcheck
}

func (h httpHandler) sendError(w http.ResponseWriter, err error, message string, statusCode int) {
	e := &httpError{
		message:    message,
		statusCode: statusCode,
		err:        err,
	}

	w.WriteHeader(e.StatusCode())
	h.encodeJSON(w, h)
}

func NewHTTPHandler(topo *Topographer) http.Handler {
	handler := httpHandler{
		topo: topo,
	}
	mux := http.NewServeMux()

	mux.HandleFunc("/", handler.HandleSelf)

	return mux
}
