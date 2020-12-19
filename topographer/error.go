package topographer

import (
	"encoding/json"
	"net/http"
)

type jsonHTTPError struct {
	Error struct {
		Message string `json:"message"`
		Context string `json:"context"`
	} `json:"error"`
}

type httpError struct {
	message    string
	err        error
	statusCode int
}

func (h *httpError) Message() string {
	if h == nil {
		return ""
	}

	return h.message
}

func (h *httpError) Err() string {
	if err := h.Unwrap(); err != nil {
		return err.Error()
	}

	return ""
}

func (h *httpError) StatusCode() int {
	if h == nil {
		return http.StatusInternalServerError
	}

	return h.statusCode
}

func (h *httpError) Unwrap() error {
	if h == nil {
		return nil
	}

	return h.err
}

func (h *httpError) Error() string {
	switch {
	case h == nil:
		return ""
	case h.err != nil && h.message != "":
		return h.message + ": " + h.err.Error()
	case h.err != nil:
		return h.message
	}

	return ""
}

func (h *httpError) MarshalJSON() ([]byte, error) {
	value := jsonHTTPError{}
	value.Error.Message = h.Message()
	value.Error.Context = h.Err()

	return json.Marshal(&value)
}
