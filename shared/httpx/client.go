package httpx

import (
	"context"
	"net/http"
	"time"
)

func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

func DoRequest(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	return client.Do(req)
}
