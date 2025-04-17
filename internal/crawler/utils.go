package crawler

import (
	"net/http"
	"time"
	"golang.org/x/time/rate"
)

// RateLimitedClient wraps an http.Client with rate limiting
type RateLimitedClient struct {
	client      *http.Client
	rateLimiter *rate.Limiter
}

func NewRateLimitedClient(requestsPerSecond float64) *RateLimitedClient {
	return &RateLimitedClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		rateLimiter: rate.NewLimiter(rate.Limit(requestsPerSecond), 1),
	}
}

func (c *RateLimitedClient) Do(req *http.Request) (*http.Response, error) {
	err := c.rateLimiter.Wait(req.Context())
	if err != nil {
		return nil, err
	}
	return c.client.Do(req)
}
