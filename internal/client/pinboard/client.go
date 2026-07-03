package pinboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	BaseURL         = "https://api.pinboard.in/v1"
	RateLimit       = 3 * time.Second
	RatePostsAll    = 5 * time.Minute
	RatePostsRecent = 1 * time.Minute

	// maxRetries429 is the number of times a request is retried after a
	// 429 response before giving up.
	maxRetries429 = 3
	// defaultRetryDelay is the base delay before retrying a 429 response;
	// it doubles on each subsequent retry unless the response carries a
	// Retry-After header.
	defaultRetryDelay = 5 * time.Second
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
	retryDelay      time.Duration
	lastRequest     time.Time
	lastPostsAll    time.Time
	lastPostsRecent time.Time
}

func NewClient(auth AuthMethod) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		auth:       auth,
		baseURL:    BaseURL,
		retryDelay: defaultRetryDelay,
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
		if elapsed := now.Sub(c.lastPostsAll); elapsed < RatePostsAll {
			time.Sleep(RatePostsAll - elapsed)
		}
		c.lastPostsAll = now
	case "posts/recent":
		if elapsed := now.Sub(c.lastPostsRecent); elapsed < RatePostsRecent {
			time.Sleep(RatePostsRecent - elapsed)
		}
		c.lastPostsRecent = now
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

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return nil, err
		}

		c.auth.Apply(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			resp.Body.Close()

			if attempt == maxRetries429 {
				return nil, fmt.Errorf("API request rate limited (status 429) after %d attempts", attempt+1)
			}

			backoff := c.retryDelay << attempt
			if secs, err := strconv.Atoi(retryAfter); err == nil && secs >= 0 {
				backoff = time.Duration(secs) * time.Second
			}

			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}

		return resp, nil
	}
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
