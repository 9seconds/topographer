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

	req.Header.Set("User-Agent", h.userAgent)

	resp, err := h.circuitBreaker.Do(ctx, func() (interface{}, error) {
		resp, err := h.client.Do(req.WithContext(ctx))

		if err != nil {
			if resp != nil {
				io.Copy(ioutil.Discard, resp.Body)
				resp.Body.Close()
			}

			return nil, err
		}

		if resp.StatusCode >= http.StatusBadRequest {
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()

			return nil, fmt.Errorf("Netloc has responded with %s", resp.Status)
		}

		return resp, err
	})

	return resp.(*http.Response), err
}

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
