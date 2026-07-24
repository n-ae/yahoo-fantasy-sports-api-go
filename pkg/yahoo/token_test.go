package yahoo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type fakeStore struct {
	mu    sync.Mutex
	saved []Token
	err   error
}

func (s *fakeStore) Save(ctx context.Context, tok Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saved = append(s.saved, tok)
	return s.err
}

func (s *fakeStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.saved)
}

// newRefreshingServers returns an API server that 401s the old token and 200s
// the new one, plus a token server that rotates to new-token/new-refresh.
func newRefreshingServers(t *testing.T) (api, token *httptest.Server) {
	t.Helper()
	token = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"access_token":"new-token","refresh_token":"new-refresh","expires_in":3600}`))
	}))
	api = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer old-token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"token_expired"}`))
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	return api, token
}

func clientForRefresh(t *testing.T, api, token *httptest.Server, opts ...Option) *Client {
	t.Helper()
	base := []Option{
		WithTokens("old-token", "old-refresh"),
		WithBaseURL(api.URL),
		WithTokenURL(token.URL),
		WithHTTPClient(&http.Client{}),
	}
	c, err := NewClient(append(base, opts...)...)
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	return c
}

func TestTokenStoreSavedOnRefresh(t *testing.T) {
	api, token := newRefreshingServers(t)
	defer api.Close()
	defer token.Close()

	store := &fakeStore{}
	c := clientForRefresh(t, api, token, WithTokenStore(store))

	if _, err := c.makeRequest(context.Background(), "x"); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if store.count() != 1 {
		t.Fatalf("Save called %d times, want 1", store.count())
	}
	got := store.saved[0]
	if got.AccessToken != "new-token" || got.RefreshToken != "new-refresh" {
		t.Errorf("saved token = %+v, want rotated new-token/new-refresh", got)
	}
	if got.ExpiresAt.IsZero() {
		t.Error("saved token ExpiresAt not set")
	}
}

// Under a single-flight refresh, only the goroutine that refreshed persists.
func TestTokenStoreSavedOnceUnderConcurrency(t *testing.T) {
	api, token := newRefreshingServers(t)
	defer api.Close()
	defer token.Close()

	store := &fakeStore{}
	c := clientForRefresh(t, api, token, WithTokenStore(store))

	const n = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			c.makeRequest(context.Background(), "x")
		}()
	}
	close(start)
	wg.Wait()

	if store.count() != 1 {
		t.Errorf("Save called %d times, want 1 (single-flight)", store.count())
	}
}

// A Save error is advisory: it must not fail the API request.
func TestTokenStoreErrorIsAdvisory(t *testing.T) {
	api, token := newRefreshingServers(t)
	defer api.Close()
	defer token.Close()

	logger := &capturingLogger{}
	store := &fakeStore{err: errors.New("disk full")}
	c := clientForRefresh(t, api, token, WithTokenStore(store), WithLogger(logger))

	body, err := c.makeRequest(context.Background(), "x")
	if err != nil {
		t.Fatalf("request should succeed despite Save error, got %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q", body)
	}
	if len(logger.msgs) == 0 {
		t.Error("expected a persistence-failure log message")
	}
}
