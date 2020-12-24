package topolib

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mercari/go-circuitbreaker"
	"golang.org/x/time/rate"
)

type httpClient struct {
	client         *http.Client
	rateLimiter    *rate.Limiter
	circuitBreaker *circuitbreaker.CircuitBreaker
}

func (h httpClient) Do(req *http.Request) (*http.Response, error) {
	var ctx context.Context
	var cancel context.CancelFunc

	if h.client.Timeout > 0 {
		ctx, cancel = context.WithTimeout(req.Context(), h.client.Timeout)
		defer cancel()
	} else {
		ctx = req.Context()
	}

	if err := h.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("cannot execute a request due to rate limiter: %w", err)
	}

	resp, err := h.circuitBreaker.Do(ctx, func() (interface{}, error) {
		return h.client.Do(req.WithContext(ctx))
	})

	return resp.(*http.Response), err
}

func NewHTTPClient(client *http.Client,
	rateLimiterInterval time.Duration,
	rateLimitBurst int) HTTPClient {
	return httpClient{
		client:         client,
		rateLimiter:    rate.NewLimiter(rate.Every(rateLimiterInterval), rateLimitBurst),
		circuitBreaker: circuitbreaker.New(nil),
	}
}
