package pinboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	BaseURL         = "https://api.pinboard.in/v1"
	RateLimit       = 3 * time.Second
	PostsAllRate    = 5 * time.Minute
	PostsRecentRate = 1 * time.Minute
)

type AuthMethod interface {
	Apply(*http.Request)
}

type BasicAuth struct {
	Username string
	Password string
}

func (a BasicAuth) Apply(req *http.Request) {
	req.SetBasicAuth(a.Username, a.Password)
}

type TokenAuth struct {
	Username string
	Token    string
}

func (a TokenAuth) Apply(req *http.Request) {
	q := req.URL.Query()
	q.Set("auth_token", fmt.Sprintf("%s:%s", a.Username, a.Token))
	req.URL.RawQuery = q.Encode()
}

type Client struct {
	httpClient      *http.Client
	auth            AuthMethod
	baseURL         string
	lastRequest     time.Time
	postsAllLast    time.Time
	postsRecentLast time.Time
}

func NewClient(auth AuthMethod) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		auth:       auth,
		baseURL:    BaseURL,
	}
}

func NewClientFromCredentials() (*Client, error) {
	creds, err := LoadCredentials()
	if err != nil {
		return nil, err
	}

	auth := TokenAuth{
		Username: creds.Username,
		Token:    creds.Token,
	}

	return NewClient(auth), nil
}

func (c *Client) WithHTTPClient(client *http.Client) *Client {
	c.httpClient = client
	return c
}

func (c *Client) rateLimit(endpoint string) {
	now := time.Now()

	switch endpoint {
	case "posts/all":
		if elapsed := now.Sub(c.postsAllLast); elapsed < PostsAllRate {
			time.Sleep(PostsAllRate - elapsed)
		}
		c.postsAllLast = now
	case "posts/recent":
		if elapsed := now.Sub(c.postsRecentLast); elapsed < PostsRecentRate {
			time.Sleep(PostsRecentRate - elapsed)
		}
		c.postsRecentLast = now
	}

	if elapsed := now.Sub(c.lastRequest); elapsed < RateLimit {
		time.Sleep(RateLimit - elapsed)
	}
	c.lastRequest = time.Now()
}

func (c *Client) makeRequest(ctx context.Context, endpoint string, params url.Values) (*http.Response, error) {
	c.rateLimit(endpoint)

	reqURL := fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	if params == nil {
		params = url.Values{}
	}
	params.Set("format", "json")

	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	c.auth.Apply(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 429 {
		resp.Body.Close()

		backoff := 5 * time.Second
		select {
		case <-time.After(backoff):
			return c.makeRequest(ctx, endpoint, params)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return resp, nil
}

func (c *Client) GetAPIToken(ctx context.Context) (string, error) {
	resp, err := c.makeRequest(ctx, "user/api_token", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Result string `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode API token response: %w", err)
	}

	return result.Result, nil
}

func (c *Client) GetSecret(ctx context.Context) (string, error) {
	resp, err := c.makeRequest(ctx, "user/secret", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Result string `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode secret response: %w", err)
	}

	return result.Result, nil
}
