package topolib

import (
	"encoding/json"
	"errors"
	"net/http"
)

var (
	ErrTopographerShutdown = errors.New("topographer instance was shutdown")
	ErrContextIsClosed     = errors.New("context is closed")
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
	if err := errors.Unwrap(h); err != nil {
		return err.Error()
	}

	return ""
}

func (h *httpError) StatusCode() int {
	if h != nil && h.statusCode != 0 {
		return h.statusCode
	}

	return http.StatusInternalServerError
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
		return h.err.Error()
	}

	return h.message
}

func (h *httpError) MarshalJSON() ([]byte, error) {
	value := jsonHTTPError{}
	value.Error.Message = h.Message()
	value.Error.Context = h.Err()

	return json.Marshal(&value)
}
