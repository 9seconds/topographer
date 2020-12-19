package topographer

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/pariz/gountries"
)

type api struct {
	ctx          context.Context
	logger       Logger
	providers    map[string]Provider
	countryQuery *gountries.Query
}

func encodeJSON(w http.ResponseWriter, data interface{}) {
	encoder := json.NewEncoder(w)

	w.Header().Add("Content-Type", "application/json")
	encoder.SetEscapeHTML(false)
	encoder.Encode(data) // nolint: errcheck
}

func sendError(w http.ResponseWriter, err error, message string, statusCode int) {
	h := &httpError{
		message:    message,
		statusCode: statusCode,
		err:        err,
	}

	w.WriteHeader(h.StatusCode())
	encodeJSON(w, h)
}
