package topolib

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/qri-io/jsonschema"
)

var handlePostRequestJSONSchema = func() *jsonschema.Schema {
	data := `{
        "type": "object",
        "required": [
            "ips"
        ],
        "additionalProperties": false,
        "properties": {
            "ips": {
                "type": "array",
                "minItems": 1,
                "items": {
                    "anyOf": [
                        {
                            "type": "string",
                            "format": "ipv4",
                            "minLength": 7,
                            "maxLength": 15
                        },
                        {
                            "type": "string",
                            "format": "ipv6",
                            "minLength": 2,
                            "maxLength": 39
                        }
                    ]
                }
            },
            "providers": {
                "type": "array",
                "items": {
                    "type": "string",
                    "minLength": 1
                }
            }
        }
    }`

	rv := &jsonschema.Schema{}
	if err := json.Unmarshal([]byte(data), rv); err != nil {
		panic(err)
	}

	return rv
}()

type handlePostRequest struct {
	IPs       []net.IP `json:"ips"`
	Providers []string `json:"providers"`
}

type handlePostResponse struct {
	Results []ResolveResult `json:"results"`
}

func (h httpHandler) handlePost(w http.ResponseWriter, req *http.Request) {
	if !strings.Contains(req.Header.Get("Content-Type"), "application/json") {
		h.sendError(w, nil, "Incorrect content type", http.StatusUnsupportedMediaType)

		return
	}

	bodyBytes, err := ioutil.ReadAll(req.Body)

	req.Body.Close()

	if err != nil {
		h.sendError(w, err, "Cannot read request body", http.StatusBadRequest)

		return
	}

	errs, err := handlePostRequestJSONSchema.ValidateBytes(req.Context(), bodyBytes)
	if err != nil {
		h.sendError(w, err, "Cannot validate body", http.StatusInternalServerError)

		return
	}

	if len(errs) > 0 {
		h.sendError(w, errs[0], "Invalid request body", http.StatusBadRequest)

		return
	}

	parsedRequest := &handlePostRequest{}
	if err := json.Unmarshal(bodyBytes, parsedRequest); err != nil {
		h.sendError(w, err, "Cannot parse request JSON", http.StatusBadRequest)

		return
	}

	resolved, err := h.topo.ResolveAll(req.Context(),
		handlePostUniqueIPs(parsedRequest.IPs),
		handlePostUniqueProviders(parsedRequest.Providers))
	if err != nil {
		h.sendError(w, err, "Cannot resolve given IPs", http.StatusInternalServerError)

		return
	}

	for i := range resolved {
		if !resolved[i].OK() {
			h.sendError(w, nil, "Cannot resolve IP address yet", http.StatusServiceUnavailable)

			return
		}
	}

	respEnvelope := handlePostResponse{
		Results: resolved,
	}

	h.encodeJSON(w, respEnvelope)
}

func handlePostUniqueIPs(ips []net.IP) []net.IP {
	uniques := map[string]bool{}

	for _, v := range ips {
		uniques[string(v)] = true
	}

	rv := make([]net.IP, 0, len(uniques))

	for k := range uniques {
		rv = append(rv, net.IP(k))
	}

	return rv
}

func handlePostUniqueProviders(names []string) []string {
	uniques := map[string]bool{}

	for _, v := range names {
		uniques[v] = true
	}

	rv := make([]string, 0, len(uniques))

	for k := range uniques {
		rv = append(rv, k)
	}

	return rv
}
