package yahoo

import (
	"context"
	"sync"
	"testing"
	"time"
)

// memCache is a minimal in-memory Cache implementation exercising the v2
// context/[]byte interface.
type memCache struct {
	mu sync.Mutex
	m  map[string][]byte
}

func (c *memCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, ok := c.m[key]
	return b, ok, nil
}

func (c *memCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = map[string][]byte{}
	}
	c.m[key] = value
	return nil
}

// A caller-supplied Cache must round-trip typed values through the generic
// cacheGet/cacheSet helpers.
func TestWithCacheRoundTrip(t *testing.T) {
	c, err := NewClient(WithTokens("t", ""), WithCache(&memCache{}))
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	ctx := context.Background()

	if _, ok := cacheGet[[]League](ctx, c, "k"); ok {
		t.Fatal("expected miss on empty cache")
	}

	cacheSet(ctx, c, "k", []League{{LeagueName: "X", NumTeams: 12}}, time.Hour)

	got, ok := cacheGet[[]League](ctx, c, "k")
	if !ok {
		t.Fatal("expected cache hit after Set")
	}
	if len(got) != 1 || got[0].LeagueName != "X" || got[0].NumTeams != 12 {
		t.Errorf("round-tripped value = %+v, want {X, 12}", got)
	}
}

// With no cache configured, the helpers are a no-op and never panic.
func TestCacheDisabledNoop(t *testing.T) {
	c, err := NewClient(WithTokens("t", ""))
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	ctx := context.Background()
	cacheSet(ctx, c, "k", []League{{LeagueName: "X"}}, time.Hour)
	if _, ok := cacheGet[[]League](ctx, c, "k"); ok {
		t.Error("disabled cache should always miss")
	}
}
