package authclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"pz1/shared/httpx"
	"pz1/shared/middleware"
)

type Client struct {
	baseURL string
	client  *http.Client
}

func New(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, client: httpx.NewClient(timeout)}
}

func (c *Client) VerifyToken(ctx context.Context, bearer string) (string, int, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/v1/auth/verify", nil)
	if err != nil {
		return "", 0, err
	}
	if bearer != "" {
		req.Header.Set("Authorization", bearer)
	}

	if v := ctx.Value(middleware.RequestIDKey()); v != nil {
		if rid, ok := v.(string); ok && rid != "" {
			req.Header.Set("X-Request-ID", rid)
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := httpx.DoRequest(ctx, c.client, req)
	if err != nil {
		return "", 0, fmt.Errorf("auth request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var body struct {
			Valid   bool   `json:"valid"`
			Subject string `json:"subject"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return "", resp.StatusCode, fmt.Errorf("invalid response body")
		}
		if body.Valid {
			return body.Subject, resp.StatusCode, nil
		}
		return "", resp.StatusCode, fmt.Errorf("invalid token")
	}

	return "", resp.StatusCode, fmt.Errorf("auth server returned %d", resp.StatusCode)
}
