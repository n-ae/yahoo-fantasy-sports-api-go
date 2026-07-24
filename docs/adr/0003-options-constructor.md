# ADR 0003: Options-Based Constructor with Typed Configuration Errors

## Status

Proposed

## Context

The current constructor is:

```go
func NewClient(apiKey, apiSecret string, db *sql.DB) *Client
```

It has several structural problems, documented in the v4 maintainability
assessment ([docs/assessments/0001](../assessments/0001-maintainable-architect-v4-assessment.md),
finding M2):

1. **It cannot report configuration errors.** It returns only `*Client`, so a
   missing token, an empty credential, or an enabled cache with a `nil`
   database cannot surface at construction time. Failures appear later, deep in
   a request. (v1.5.0 added a runtime guard against the nil-DB panic, but the
   misconfiguration is still only discovered on first use.)
2. **A database is mandatory.** Persistence is a policy, not an inherent
   requirement of making Yahoo requests. A CLI that only reads a roster must
   still pass a `*sql.DB`.
3. **Configuration is read from hidden environment variables.**
   `NewClient` silently reads `YAHOO_CONSUMER_KEY`, `YAHOO_CONSUMER_SECRET`,
   `YAHOO_ACCESS_TOKEN`, `YAHOO_REFRESH_TOKEN`, `YAHOO_BASE_URL`, and
   `YAHOO_ENABLE_CACHE`. Behavior is not visible from the arguments, tests can
   interfere through process-global state, and a caller cannot distinguish an
   explicit empty value from an environment fallback.
4. **Dependencies are not injectable.** The `*http.Client`, the cache, and the
   token handling are all constructed internally, which blocks `httptest`
   transports, custom timeouts, tracing, and deterministic unit tests.

The forces: this is a solo-maintained library that wants a small, stable public
API and easy testing, without adopting a heavy configuration framework.

## Decision

**Introduce a functional-options constructor that validates configuration and
returns an error, and move environment reading to an explicit, opt-in helper.**

```go
func NewClient(opts ...Option) (*Client, error)

type Option func(*config) error

func WithCredentials(key, secret string) Option
func WithTokens(access, refresh string) Option
func WithHTTPClient(HTTPDoer) Option      // inject *http.Client or a mock
func WithBaseURL(string) Option
func WithCache(Cache) Option              // optional; no cache by default
func WithLogger(Logger) Option            // replaces direct stdout logging
func FromEnv() Option                     // explicit, opt-in env loading
```

Supporting interfaces (small, dependency-free):

```go
type HTTPDoer interface{ Do(*http.Request) (*http.Response, error) }

type Cache interface {
    Get(ctx context.Context, key string) ([]byte, bool, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}
```

Construction validates and fails fast:

- unknown/invalid option → error;
- credentials or tokens required but empty → error;
- `WithCache` requires a non-nil implementation.

The existing `NewClient(apiKey, apiSecret string, db *sql.DB) *Client` is kept
as a **thin, deprecated wrapper** over the new constructor for one minor-version
cycle, so current callers keep compiling.

## Consequences

### Positive

- **Errors surface at construction**, not mid-request. Misconfiguration is a
  compile-adjacent, testable failure.
- **No mandatory database.** Read-only CLIs construct a client with just
  credentials and tokens.
- **Testable transport.** `WithHTTPClient(httptest…)` enables deterministic
  unit tests of request construction, status handling, and refresh — exactly
  the highest-risk boundary the assessment flags as under-tested.
- **Explicit configuration.** `FromEnv()` makes environment loading a visible
  choice; multi-account processes stop fighting over global env state.
- **Enables ADR-0002.** An injected `Cache` interface is what lets SQLite (and
  cgo) leave the SDK core.

### Negative

- **Two constructors during the deprecation window**, which is mild API
  surface duplication.
- **Callers eventually migrate** from the positional constructor to options.
- Slightly more code than a single struct literal.

### Mitigation Strategies

- Keep the deprecated `NewClient(key, secret, db)` wrapper for one minor cycle
  with a `// Deprecated:` doc comment pointing at the options constructor and a
  removal target version.
- Provide a copy-paste migration snippet in `CHANGELOG.md` and README.
- Default behavior with no options should be sensible (30s timeout HTTP client,
  no cache, official base/token URLs) so common cases stay short.

## Alternatives Considered

1. **Config struct: `NewClient(Config) (*Client, error)`.**
   - Pros: simple, explicit, easy to read.
   - Cons: zero-value ambiguity (is an empty field "unset" or "explicitly
     empty"?), and awkward to extend without either breaking the struct or
     adding pointers everywhere. Functional options express "unset" cleanly.
     Reasonable second choice.
2. **Keep the positional constructor, just add an `error` return.**
   - Pros: minimal change.
   - Cons: still mandates a DB, still can't inject transport/cache, still hides
     env behavior. Doesn't address the core problems. Rejected.
3. **Builder pattern (`NewBuilder().WithX().Build()`).**
   - Pros: fluent.
   - Cons: more machinery than functional options for no added expressiveness in
     Go; less idiomatic. Rejected.

## Implementation Details

- Add `config` (private) + `Option` and the `With*` / `FromEnv` functions.
- `NewClient(opts...)` builds `config`, applies options in order, validates,
  and constructs `*Client`.
- Move the six `os.Getenv` reads out of the constructor into `FromEnv()`.
- Replace the direct `fmt.Printf` refresh log (assessment finding m3) with an
  injected `Logger` that defaults to a no-op.
- Deprecated shim:
  ```go
  // Deprecated: use NewClient(WithCredentials(...), WithTokens(...), WithCache(...)).
  // Removed in a future release.
  func NewClientLegacy(apiKey, apiSecret string, db *sql.DB) *Client {
      c, _ := NewClient(WithCredentials(apiKey, apiSecret), FromEnv(), withSQLDB(db))
      return c
  }
  ```
- Breaking change: the *signature* of `NewClient` changes. This is the main
  reason to schedule it deliberately (major, or a minor with the old name kept
  as a wrapper under a new name). Sequence with [ADR-0002](./0002-separate-sdk-from-application.md).

## Examples

Read-only CLI, no database, fully testable:

```go
client, err := yahoo.NewClient(
    yahoo.WithCredentials(key, secret),
    yahoo.WithTokens(access, refresh),
)
if err != nil {
    log.Fatal(err) // misconfig fails here, not mid-request
}
```

Deterministic unit test:

```go
srv := httptest.NewServer(handler)
client, _ := yahoo.NewClient(
    yahoo.WithTokens("t", ""),
    yahoo.WithBaseURL(srv.URL),
    yahoo.WithHTTPClient(srv.Client()),
)
```

Explicit environment loading (opt-in):

```go
client, err := yahoo.NewClient(yahoo.FromEnv())
```

## Success Metrics

- Constructing with invalid config returns a non-nil error in tests.
- `pkg/yahoo` request-path tests run against `httptest` without touching the
  network or process env.
- No `os.Getenv` calls remain in the default construction path.
- The refresh path no longer writes to stdout.

## References

- [docs/assessments/0001 — Maintainability Assessment](../assessments/0001-maintainable-architect-v4-assessment.md) (findings M2, M3, m3)
- Functional options: Dave Cheney, "Functional options for friendly APIs"
- `golang.org/x/oauth2` for the eventual token-source direction

## Related Decisions

- [ADR-0002: Separate the Reusable SDK from the Bundled Application](./0002-separate-sdk-from-application.md)
  — the `Cache`/`HTTPDoer` seams introduced here are what allow SQLite to leave
  the SDK core.

## Notes

- A later ADR should cover replacing manual token refresh with an
  `oauth2.TokenSource` plus single-flight and a persistence callback; this ADR
  intentionally scopes only the constructor and dependency injection.
- Open question: whether `FromEnv()` should be strict (error on missing
  required vars) or lenient (fill what it finds). Leaning strict for the
  required credentials/tokens.

---

**Decision Date**: 2026-07-24
**Participants**: Maintainer; maintainable-architect-v4 review
**Outcome**: Proposed — replace the positional, error-free, DB-mandatory
constructor with a validating functional-options constructor, move environment
loading behind an explicit `FromEnv()` option, and inject HTTP/cache/logger
dependencies.
