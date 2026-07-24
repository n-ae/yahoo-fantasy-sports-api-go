package yahoo

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	defaultBaseURL  = "https://fantasysports.yahooapis.com/fantasy/v2"
	defaultTokenURL = "https://api.login.yahoo.com/oauth2/get_token"
)

// HTTPDoer is the subset of *http.Client the client needs. Injecting it enables
// custom transports, timeouts, tracing, and httptest servers in tests.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Cache is a pluggable response cache. Enable the bundled SQLite implementation
// with WithSQLiteCache, or supply your own (in-memory, Redis, etc.) with
// WithCache. Get reports ok=false for a miss or expired entry.
type Cache interface {
	Get(ctx context.Context, key string) (value []byte, ok bool, err error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// Logger receives advisory diagnostic messages (e.g. retry notices). The
// default is a no-op; supply one with WithLogger to observe client internals.
type Logger interface {
	Printf(format string, args ...interface{})
}

type noopLogger struct{}

func (noopLogger) Printf(string, ...interface{}) {}

// Token holds the credentials produced by a token refresh. ExpiresAt is derived
// from the token endpoint's expires_in.
type Token struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// TokenStore persists tokens after a successful refresh so they survive process
// restarts (Yahoo rotates refresh tokens, so an in-memory-only client can lock
// itself out after a restart). See docs/adr/0004-token-persistence.md.
//
// Save is best-effort: an error is logged via the Logger but does not fail the
// API request, since the rotated token is already valid for this process.
// Loading initial tokens is the caller's responsibility (pass them via
// WithTokens at construction).
type TokenStore interface {
	Save(ctx context.Context, tok Token) error
}

type config struct {
	apiKey       string
	apiSecret    string
	accessToken  string
	refreshToken string
	baseURL      string
	tokenURL     string
	httpClient   HTTPDoer
	cache        Cache
	cacheEnabled bool
	logger       Logger
	tokenStore   TokenStore
	retry        RetryPolicy
}

func defaultConfig() config {
	return config{
		baseURL:    defaultBaseURL,
		tokenURL:   defaultTokenURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     noopLogger{},
		retry:      defaultRetryPolicy(),
	}
}

// Option configures a Client built with NewClient. Explicit options
// take precedence over FromEnv regardless of order.
type Option func(*config) error

// WithCredentials sets the OAuth consumer key and secret used to refresh tokens.
func WithCredentials(key, secret string) Option {
	return func(c *config) error {
		c.apiKey = key
		c.apiSecret = secret
		return nil
	}
}

// WithTokens sets the initial access and refresh tokens.
func WithTokens(access, refresh string) Option {
	return func(c *config) error {
		c.accessToken = access
		c.refreshToken = refresh
		return nil
	}
}

// WithHTTPClient injects the HTTP client used for API and token requests.
func WithHTTPClient(d HTTPDoer) Option {
	return func(c *config) error {
		if d == nil {
			return fmt.Errorf("yahoo: WithHTTPClient requires a non-nil HTTPDoer")
		}
		c.httpClient = d
		return nil
	}
}

// WithBaseURL overrides the Yahoo Fantasy API base URL (useful for testing).
func WithBaseURL(u string) Option {
	return func(c *config) error {
		if u == "" {
			return fmt.Errorf("yahoo: WithBaseURL requires a non-empty URL")
		}
		c.baseURL = u
		return nil
	}
}

// WithTokenURL overrides the OAuth token endpoint (useful for testing).
func WithTokenURL(u string) Option {
	return func(c *config) error {
		if u == "" {
			return fmt.Errorf("yahoo: WithTokenURL requires a non-empty URL")
		}
		c.tokenURL = u
		return nil
	}
}

// WithCache enables response caching backed by the supplied Cache.
func WithCache(cache Cache) Option {
	return func(c *config) error {
		if cache == nil {
			return fmt.Errorf("yahoo: WithCache requires a non-nil Cache")
		}
		c.cache = cache
		c.cacheEnabled = true
		return nil
	}
}

// WithSQLiteCache enables response caching backed by the bundled SQLite cache.
// The caller owns the database and the yahoo_api_cache table schema.
func WithSQLiteCache(db *sql.DB) Option {
	return func(c *config) error {
		if db == nil {
			return fmt.Errorf("yahoo: WithSQLiteCache requires a non-nil *sql.DB")
		}
		c.cache = &apiCache{db: db}
		c.cacheEnabled = true
		return nil
	}
}

// WithLogger sets an advisory logger for client diagnostics.
func WithLogger(l Logger) Option {
	return func(c *config) error {
		if l == nil {
			return fmt.Errorf("yahoo: WithLogger requires a non-nil Logger")
		}
		c.logger = l
		return nil
	}
}

// WithTokenStore persists tokens after each successful refresh (see TokenStore).
func WithTokenStore(s TokenStore) Option {
	return func(c *config) error {
		if s == nil {
			return fmt.Errorf("yahoo: WithTokenStore requires a non-nil TokenStore")
		}
		c.tokenStore = s
		return nil
	}
}

// WithRetryPolicy overrides the default policy for retrying transient GET
// failures (429 and 5xx). See RetryPolicy.
func WithRetryPolicy(p RetryPolicy) Option {
	return func(c *config) error {
		if err := p.validate(); err != nil {
			return err
		}
		c.retry = p
		return nil
	}
}

// FromEnv fills any still-unset credentials, tokens, and base URL from the
// standard environment variables, and enables caching if YAHOO_ENABLE_CACHE is
// "true". Explicit With* options take precedence: FromEnv only fills gaps.
//
//	YAHOO_CONSUMER_KEY, YAHOO_CONSUMER_SECRET,
//	YAHOO_ACCESS_TOKEN, YAHOO_REFRESH_TOKEN,
//	YAHOO_BASE_URL, YAHOO_ENABLE_CACHE
func FromEnv() Option {
	return func(c *config) error {
		if c.apiKey == "" {
			c.apiKey = os.Getenv("YAHOO_CONSUMER_KEY")
		}
		if c.apiSecret == "" {
			c.apiSecret = os.Getenv("YAHOO_CONSUMER_SECRET")
		}
		if c.accessToken == "" {
			c.accessToken = os.Getenv("YAHOO_ACCESS_TOKEN")
		}
		if c.refreshToken == "" {
			c.refreshToken = os.Getenv("YAHOO_REFRESH_TOKEN")
		}
		if v := os.Getenv("YAHOO_BASE_URL"); v != "" {
			c.baseURL = v
		}
		if os.Getenv("YAHOO_ENABLE_CACHE") == "true" {
			c.cacheEnabled = true
		}
		return nil
	}
}

// NewClient builds a Client from functional options, validating the resulting
// configuration and returning an error for invalid combinations.
func NewClient(opts ...Option) (*Client, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	if cfg.httpClient == nil {
		return nil, fmt.Errorf("yahoo: HTTP client is required")
	}
	if cfg.cacheEnabled && cfg.cache == nil {
		return nil, fmt.Errorf("yahoo: caching enabled but no cache configured (use WithCache or WithSQLiteCache)")
	}
	if cfg.logger == nil {
		cfg.logger = noopLogger{}
	}

	return &Client{
		apiKey:       cfg.apiKey,
		apiSecret:    cfg.apiSecret,
		accessToken:  cfg.accessToken,
		refreshToken: cfg.refreshToken,
		httpClient:   cfg.httpClient,
		baseURL:      cfg.baseURL,
		tokenURL:     cfg.tokenURL,
		cache:        cfg.cache,
		cacheEnabled: cfg.cacheEnabled,
		logger:       cfg.logger,
		tokenStore:   cfg.tokenStore,
		retry:        cfg.retry,
	}, nil
}
