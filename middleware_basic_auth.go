package main

import (
	"crypto/subtle"
	"net/http"
)

type basicAuthMiddleware struct {
	handler  http.Handler
	user     []byte
	password []byte
}

func (b *basicAuthMiddleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	user, pass, _ := req.BasicAuth()

	userBytes := []byte(user)
	passBytes := []byte(pass)

	if subtle.ConstantTimeCompare(b.user, userBytes)+subtle.ConstantTimeCompare(b.password, passBytes) == 2 {
		b.handler.ServeHTTP(w, req)

		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	http.Error(w, "Authentication is required", http.StatusUnauthorized)
}
