package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/9seconds/topographer/providers"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func MakeServer(set *providers.ProviderSet) *chi.Mux {
	router := chi.NewRouter()

	ctxProviderSet := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "providers", set)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	router.Use(middleware.StripSlashes)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(middleware.Recoverer)
	router.Use(middleware.RealIP)
	router.Use(middleware.SetHeader("Content-Type", "application/json"))
	router.Use(ctxProviderSet)

	router.Get("/", selfResolveIP)
	router.Post("/", resolveIPs)
	router.Get("/info", providerInfo)

	return router
}

func abort(w http.ResponseWriter, code int, message string) {
	msg, _ := json.Marshal(map[string]string{"error": message})
	http.Error(w, string(msg), code)
}
