package pinboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	auth := TokenAuth{Username: "test", Token: "token123"}
	client := NewClient(auth)

	if client == nil {
		t.Fatal("expected client to be created")
	}

	if client.baseURL != BaseURL {
		t.Errorf("expected baseURL %s, got %s", BaseURL, client.baseURL)
	}
}

func TestTokenAuth(t *testing.T) {
	auth := TokenAuth{Username: "testuser", Token: "testtoken"}

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	auth.Apply(req)

	expected := "testuser:testtoken"
	if got := req.URL.Query().Get("auth_token"); got != expected {
		t.Errorf("expected auth_token %s, got %s", expected, got)
	}
}

func TestBasicAuth(t *testing.T) {
	auth := BasicAuth{Username: "testuser", Password: "testpass"}

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	auth.Apply(req)

	username, password, ok := req.BasicAuth()
	if !ok {
		t.Fatal("expected basic auth to be set")
	}

	if username != "testuser" || password != "testpass" {
		t.Errorf("expected testuser:testpass, got %s:%s", username, password)
	}
}

func TestMakeRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "json" {
			t.Errorf("expected format=json in query params")
		}

		if r.URL.Query().Get("auth_token") != "test:token123" {
			t.Errorf("expected auth_token=test:token123 in query params")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	auth := TokenAuth{Username: "test", Token: "token123"}
	client := NewClient(auth)
	client.baseURL = server.URL

	resp, err := client.makeRequest(context.Background(), "test/endpoint", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestMakeRequestRateLimit429(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	auth := TokenAuth{Username: "test", Token: "token123"}
	client := NewClient(auth)
	client.baseURL = server.URL
	client.retryDelay = 10 * time.Millisecond

	resp, err := client.makeRequest(context.Background(), "test/endpoint", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if callCount != 2 {
		t.Errorf("expected 2 calls (429 then success), got %d", callCount)
	}
}

func TestMakeRequestRateLimit429Exhausted(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	auth := TokenAuth{Username: "test", Token: "token123"}
	client := NewClient(auth)
	client.baseURL = server.URL
	client.retryDelay = time.Millisecond

	_, err := client.makeRequest(context.Background(), "test/endpoint", nil)
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("expected rate limited error, got: %v", err)
	}

	wantCalls := maxRetries429 + 1
	if callCount != wantCalls {
		t.Errorf("expected %d calls (initial + %d retries), got %d", wantCalls, maxRetries429, callCount)
	}
}

func TestMakeRequestRateLimit429RetryAfter(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	auth := TokenAuth{Username: "test", Token: "token123"}
	client := NewClient(auth)
	client.baseURL = server.URL
	// A Retry-After of 0 seconds must override the configured delay; if it
	// does not, this test times out rather than finishing instantly.
	client.retryDelay = time.Hour

	start := time.Now()
	resp, err := client.makeRequest(context.Background(), "test/endpoint", nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if callCount != 2 {
		t.Errorf("expected 2 calls (429 then success), got %d", callCount)
	}
	if elapsed > 10*time.Second {
		t.Errorf("Retry-After: 0 should retry immediately, took %v", elapsed)
	}
}

func TestMakeRequestRateLimit429ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	auth := TokenAuth{Username: "test", Token: "token123"}
	client := NewClient(auth)
	client.baseURL = server.URL
	client.retryDelay = time.Hour

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := client.makeRequest(ctx, "test/endpoint", nil)
		done <- err
	}()
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error from canceled context, got nil")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("makeRequest did not return promptly after context cancellation")
	}
}

func TestGetAPIToken(t *testing.T) {
	expectedToken := "test:ABCDEF123456"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/user/api_token") {
			t.Errorf("expected path to end with /user/api_token, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "` + expectedToken + `"}`))
	}))
	defer server.Close()

	auth := TokenAuth{Username: "test", Token: "token123"}
	client := NewClient(auth)
	client.baseURL = server.URL

	token, err := client.GetAPIToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token != expectedToken {
		t.Errorf("expected token %s, got %s", expectedToken, token)
	}
}

func TestWithHTTPClient(t *testing.T) {
	auth := TokenAuth{Username: "test", Token: "token123"}
	customClient := &http.Client{Timeout: 10 * time.Second}

	client := NewClient(auth).WithHTTPClient(customClient)

	if client.httpClient != customClient {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestGetAllPostsWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		if len(query["tag"]) != 2 || query["tag"][0] != "tag1" || query["tag"][1] != "tag2" {
			t.Errorf("expected tag params: tag1,tag2, got %v", query["tag"])
		}
		if query.Get("start") != "10" {
			t.Errorf("expected start=10, got %s", query.Get("start"))
		}
		if query.Get("results") != "50" {
			t.Errorf("expected results=50, got %s", query.Get("results"))
		}
		if query.Get("meta") != "yes" {
			t.Errorf("expected meta=yes, got %s", query.Get("meta"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	auth := TokenAuth{Username: "test", Token: "token123"}
	client := NewClient(auth)
	client.baseURL = server.URL

	opts := &GetAllPostsOptions{
		Tag:     []string{"tag1", "tag2"},
		Start:   10,
		Results: 50,
		Meta:    true,
	}

	_, err := client.GetAllPosts(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddPostWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		if query.Get("url") != "https://example.com" {
			t.Errorf("expected url=https://example.com, got %s", query.Get("url"))
		}
		if query.Get("description") != "Test Bookmark" {
			t.Errorf("expected description=Test Bookmark, got %s", query.Get("description"))
		}
		if query.Get("extended") != "Extended description" {
			t.Errorf("expected extended description, got %s", query.Get("extended"))
		}
		if query.Get("tags") != "tag1 tag2" {
			t.Errorf("expected tags, got %s", query.Get("tags"))
		}
		if query.Get("replace") != "no" {
			t.Errorf("expected replace=no, got %s", query.Get("replace"))
		}
		if query.Get("shared") != "no" {
			t.Errorf("expected shared=no, got %s", query.Get("shared"))
		}
		if query.Get("toread") != "yes" {
			t.Errorf("expected toread=yes, got %s", query.Get("toread"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result_code": "done"}`))
	}))
	defer server.Close()

	auth := TokenAuth{Username: "test", Token: "token123"}
	client := NewClient(auth)
	client.baseURL = server.URL

	replace := false
	shared := false
	toread := true

	opts := &AddPostOptions{
		Extended: "Extended description",
		Tags:     "tag1 tag2",
		Replace:  &replace,
		Shared:   &shared,
		ToRead:   &toread,
	}

	err := client.AddPost(context.Background(), "https://example.com", "Test Bookmark", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetRecentPostsWithMultipleTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		if len(query["tag"]) != 2 || query["tag"][0] != "tag1" || query["tag"][1] != "tag2" {
			t.Errorf("expected tag params: tag1,tag2, got %v", query["tag"])
		}
		if query.Get("count") != "25" {
			t.Errorf("expected count=25, got %s", query.Get("count"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"posts": []}`))
	}))
	defer server.Close()

	auth := TokenAuth{Username: "test", Token: "token123"}
	client := NewClient(auth)
	client.baseURL = server.URL

	_, err := client.GetRecentPosts(context.Background(), 25, []string{"tag1", "tag2"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRateLimitFreshClientDoesNotWait(t *testing.T) {
	client := NewClient(TokenAuth{Username: "test", Token: "token123"})

	start := time.Now()
	if err := client.rateLimit(context.Background(), "posts/all"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("fresh client should not wait, took %v", elapsed)
	}

	if client.lastPostsAll.IsZero() || client.lastRequest.IsZero() {
		t.Error("expected timestamps to be recorded")
	}
}

func TestRateLimitRecordsTimeAfterWait(t *testing.T) {
	client := NewClient(TokenAuth{Username: "test", Token: "token123"})

	// Backdate the last posts/all request so ~100ms of its interval remains.
	wait := 100 * time.Millisecond
	client.lastPostsAll = time.Now().Add(-RatePostsAll + wait)
	client.lastRequest = client.lastPostsAll

	start := time.Now()
	if err := client.rateLimit(context.Background(), "posts/all"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if elapsed := time.Since(start); elapsed < wait-10*time.Millisecond {
		t.Errorf("expected to wait ~%v, only waited %v", wait, elapsed)
	}

	// The timestamp must reflect when the request was released (after the
	// wait), not when rateLimit was called; otherwise the next interval is
	// measured from too early and under-waits.
	if recorded := client.lastPostsAll; recorded.Before(start.Add(wait - 10*time.Millisecond)) {
		t.Errorf("lastPostsAll recorded pre-wait: %v is before %v", recorded, start.Add(wait))
	}
}

func TestRateLimitContextCanceled(t *testing.T) {
	client := NewClient(TokenAuth{Username: "test", Token: "token123"})

	// Force a pending wait of the full posts/all interval (5 minutes).
	mark := time.Now()
	client.lastPostsAll = mark
	client.lastRequest = mark

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := client.rateLimit(ctx, "posts/all")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if elapsed > time.Second {
		t.Errorf("canceled context should return promptly, took %v", elapsed)
	}

	// A wait that was aborted must not count as a released request.
	if !client.lastPostsAll.Equal(mark) || !client.lastRequest.Equal(mark) {
		t.Error("timestamps should not be updated when the wait is aborted")
	}
}

func TestMakeRequestRateLimitContextCanceled(t *testing.T) {
	client := NewClient(TokenAuth{Username: "test", Token: "token123"})
	client.lastRequest = time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := client.makeRequest(ctx, "posts/get", nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if elapsed > time.Second {
		t.Errorf("canceled context should abort the rate-limit wait promptly, took %v", elapsed)
	}
}
