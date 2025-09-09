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

	if elapsed < 5*time.Second {
		t.Errorf("expected at least 5s delay for 429 retry, got %v", elapsed)
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
