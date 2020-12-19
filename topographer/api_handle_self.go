package topographer

import (
	"net"
	"net/http"
)

type HandlerSelfResponse struct {
	Result *resolveResult `json:"result"`
}

func (a *api) HandlerSelf(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" && req.Method != "HEAD" {
        sendError(w, nil, "This HTTP method is not allowed", http.StatusMethodNotAllowed)

		return
	}

	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
        sendError(w, err, "Cannot detect your IP address", 0)

		return
	}

	ipAddr := net.ParseIP(host)
	if ipAddr == nil {
        sendError(w, nil, "Address was detected incorrectly", 0)

		return
	}

	resolved, err := a.Resolve([]net.IP{ipAddr}, nil)
	if err != nil {
        sendError(w, err, "Cannot resolve IP address", 0)

		return
	}

	encodeJSON(w, HandlerSelfResponse{Result: &resolved[0]})
}
