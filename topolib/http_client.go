package topolib

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/mercari/go-circuitbreaker"
	"golang.org/x/time/rate"
)

type httpClient struct {
	userAgent      string
	client         *http.Client
	rateLimiter    *rate.Limiter
	circuitBreaker *circuitbreaker.CircuitBreaker
}

func (h httpClient) Do(req *http.Request) (*http.Response, error) {
	if h.client.Timeout > 0 {
		ctx, _ := context.WithTimeout(req.Context(), h.client.Timeout) // nolint: govet
		req = req.WithContext(ctx)
	}

	ctx := req.Context()

	req.Header.Set("User-Agent", h.userAgent)

	resp, err := h.circuitBreaker.Do(ctx, func() (interface{}, error) {
		resp, err := h.client.Do(req.WithContext(ctx))

		if err := h.rateLimiter.Wait(ctx); err != nil {
			return nil, circuitbreaker.Ignore(fmt.Errorf("rate limited: %w", err))
		}

		if err != nil {
			if resp != nil {
				io.Copy(ioutil.Discard, resp.Body) // nolint: errcheck
				resp.Body.Close()
			}

			return nil, err
		}

		if resp.StatusCode >= http.StatusBadRequest {
			io.Copy(ioutil.Discard, resp.Body) // nolint: errcheck
			resp.Body.Close()

			return nil, fmt.Errorf("netloc has responded with %s", resp.Status)
		}

		return resp, err
	})

	if resp == nil {
		return nil, err
	}

	return resp.(*http.Response), err
}

// NewHTTPClient prepares a new HTTP client, wraps it with rate limiter,
// circuit breaker, sets a user agent etc.
//
// Please see https://pkg.go.dev/golang.org/x/time/rate to get a meaning
// of rate limiter parameters.
func NewHTTPClient(client *http.Client,
	userAgent string,
	rateLimiterInterval time.Duration,
	rateLimitBurst int) HTTPClient {
	return httpClient{
		userAgent:      userAgent,
		client:         client,
		rateLimiter:    rate.NewLimiter(rate.Every(rateLimiterInterval), rateLimitBurst),
		circuitBreaker: circuitbreaker.New(nil),
	}
}
