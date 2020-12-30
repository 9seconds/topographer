package topolib

import (
	"encoding/json"
	"net/http"
)

type httpHandler struct {
	topo *Topographer
}

func (h httpHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGet(w, req)
	case http.MethodPost:
		h.handlePost(w, req)
	default:
		h.sendError(w, nil, "Method is not allowed", http.StatusMethodNotAllowed)
	}
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
	h.encodeJSON(w, e)
}
