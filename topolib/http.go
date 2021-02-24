package topolib

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

type httpHandler struct {
	topo *Topographer
}

func (h httpHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := strings.Trim(req.URL.Path, "/")

	switch req.Method {
	case http.MethodGet, http.MethodHead:
		switch path {
		case "":
			h.handleGetResolve(w, req)
		case "stats":
			h.handleGetStats(w, req)
		default:
			if ipAddr := net.ParseIP(path); ipAddr != nil {
				h.handleGetIP(w, req, ipAddr)
			} else {
				h.sendError(w, nil, "URL not found", http.StatusNotFound)
			}
		}
	case http.MethodPost:
		switch path {
		case "":
			h.handlePost(w, req)
		default:
			h.sendError(w, nil, "URL not found", http.StatusNotFound)
		}
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
