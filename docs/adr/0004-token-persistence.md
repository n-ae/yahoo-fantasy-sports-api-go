# ADR 0004: OAuth Token Persistence

## Status

Accepted (implemented additively in v1.8.0)

## Context

The client refreshes an expired access token by exchanging the refresh token
(see `refreshIfStale`). As of v1.6.1 that refresh is single-flight,
context-aware, and race-free. One gap remains from the maintainability
assessment ([docs/assessments/0001](../assessments/0001-maintainable-architect-v4-assessment.md),
finding 3.1):

**Refreshed tokens live only in memory.** After a process restart the
application reloads whatever tokens it started with. This is actively dangerous
because Yahoo *rotates the refresh token* on many refreshes: once the client has
refreshed, the original refresh token the app has on disk may already be
invalid, and the app cannot authenticate until a human re-runs the OAuth flow.

For an unattended, long-running service this turns a routine token refresh into
an eventual outage.

The forces:

- This is a solo-maintained library that values minimal dependencies.
- The existing manual refresh already handles the hard parts (single-flight,
  context, thread-safety). Whatever we add must not duplicate that.
- Persistence is inherently the *application's* concern (where and how to
  store secrets), so the SDK should expose a seam, not a storage implementation.

## Decision

**Add a small, dependency-free `TokenStore` hook that the client calls after a
successful refresh**, so the application can persist rotated tokens. Wire it
through the options constructor from [ADR-0003](./0003-options-constructor.md).

```go
type Token struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
}

type TokenStore interface {
    Save(ctx context.Context, tok Token) error
}

func WithTokenStore(TokenStore) Option
```

Behavior:

- After `refreshIfStale` successfully rotates the token, it calls
  `TokenStore.Save` **outside** the token lock (values captured under the lock)
  so store I/O never blocks token reads.
- A single-flight *skip* (another goroutine already refreshed) does **not** call
  `Save` — only the goroutine that actually refreshed persists.
- **`Save` errors are advisory**: they are logged via the `Logger` and do not
  fail the API request. The rotated token is valid for this process either way;
  failing the user's request because a disk write hiccuped would be worse. The
  trade-off (a persistence failure can still lose a rotation) is documented so
  callers who need stronger guarantees can make `Save` durable/retrying.

Loading initial tokens stays the caller's responsibility: they read from their
store and pass `WithTokens(access, refresh)` at construction. The SDK does not
read secrets on its own.

### We deliberately do NOT adopt `golang.org/x/oauth2`

The v2 roadmap floated `WithTokenSource(oauth2.TokenSource)`. On implementation
we reject it:

- The existing manual refresh already provides single-flight, context
  propagation, and thread safety — the things `oauth2` would give us.
- Adopting `oauth2.TokenSource` would add a dependency and a *second*,
  parallel refresh mechanism to reconcile with the current one.
- `oauth2`'s `ReuseTokenSource` still has no persistence story; we'd be adding
  a `TokenStore` wrapper anyway.

The marginal benefit does not justify the dependency and duplication. A plain
persistence hook fixes the actual gap (3.1) with far less surface area — which
is the maintainability-first call.

## Consequences

### Positive

- **Restart-safe under refresh-token rotation.** Applications can persist the
  rotated tokens and reload them, eliminating the "works until the first
  restart after a refresh" outage.
- **No new dependency**; the SDK stays pure-Go for `pkg/yahoo` consumers.
- **Storage stays the app's choice** (file, DB, secret manager) behind a
  one-method interface.
- Composes cleanly with the existing single-flight refresh — no second refresh
  path.

### Negative

- `Save` is best-effort by default, so a persistence failure can still lose a
  rotation. Documented; callers needing stronger guarantees implement a durable
  `Save`.
- The SDK still does not *load* tokens; callers must wire that themselves. This
  is intentional (secrets are the app's concern) but is one more integration
  step.

### Mitigation Strategies

- Document the advisory-`Save` semantics on `WithTokenStore` and in the README.
- Emit a `Logger` message on `Save` failure so it is observable.
- Provide a short "restart-safe tokens" example (load → `WithTokens` →
  `WithTokenStore` → save on refresh).

## Alternatives Considered

1. **`oauth2.TokenSource` integration.** Rejected — see above (dependency +
   duplicate refresh path for little gain).
2. **Fail the request on `Save` error.** Rejected — the token is already valid
   in memory; failing the user's call is worse than a logged persistence miss.
   Callers who disagree can block/retry inside their `Save`.
3. **A full `Load`+`Save` store the client reads at construction.** Deferred —
   loading secrets is the application's job; keeping the SDK write-only avoids
   baking storage/secret assumptions into the library. Can be revisited if a
   clear need appears.
4. **A bare callback `func(Token) error` instead of an interface.** Close call;
   the interface is marginally more discoverable and lets implementations carry
   state (a DB handle). Minor preference.

## Implementation Details

- `Token`, `TokenStore`, and `WithTokenStore` live in `options.go` beside the
  other seams; the `Client` gains a `tokenStore TokenStore` field.
- `refreshIfStale` is split so the locked critical section returns the new token
  values and a "did refresh" flag; `Save` runs after the lock is released and
  only when a refresh actually happened.
- `ExpiresAt` is derived from the token response's `expires_in`.
- No exported behavior changes when `WithTokenStore` is not supplied.

## Examples

Restart-safe tokens:

```go
tok := myStore.Load()          // app-owned load
client, _ := yahoo.NewClientWithOptions(
    yahoo.WithCredentials(key, secret),
    yahoo.WithTokens(tok.AccessToken, tok.RefreshToken),
    yahoo.WithTokenStore(myStore), // Save called on each refresh
)
```

```go
type myStore struct{ /* db handle */ }

func (s *myStore) Save(ctx context.Context, t yahoo.Token) error {
    // persist t.AccessToken, t.RefreshToken, t.ExpiresAt
    return nil
}
```

## Success Metrics

- A test with a fake store shows `Save` is called exactly once per real refresh,
  with the rotated tokens, and not called on a single-flight skip.
- A `Save` error does not fail the API request (logged instead).
- No dependency added to `go.mod`.

## References

- [docs/assessments/0001 — Maintainability Assessment](../assessments/0001-maintainable-architect-v4-assessment.md) (finding 3.1)
- [ADR-0003: Options-Based Constructor](./0003-options-constructor.md)
- [v2 roadmap, Phase B](../v2-roadmap.md)

## Related Decisions

- [ADR-0003](./0003-options-constructor.md) — supplies the `WithTokenStore`
  option seam.

## Notes

- If a future need for `oauth2` interop appears (e.g. sharing a token source
  with other Yahoo APIs), revisit alternative 1; the `TokenStore` hook does not
  preclude it.

---

**Decision Date**: 2026-07-24
**Participants**: Maintainer; maintainable-architect-v4 review
**Outcome**: Accepted — add a dependency-free `TokenStore` persistence hook
called after a successful refresh; do not adopt `golang.org/x/oauth2`.
