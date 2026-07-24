# ADR 0007: Deliberately Declined Hardening (Solo, Read-Only Scope)

## Status

Accepted

## Context

Three external maintainability reviews (see `docs/assessments/0001`–`0003`) have
graded this module and produced a large "hardening backlog." Those reviews are
valuable and their factual findings are almost all correct, but they are framed
for a **multi-user, unattended, production SaaS** consuming the SDK. Their
severity scores assume that lens.

This project is different, and intentionally so:

- it is **solo-maintained**;
- it is a **read-only** Yahoo Fantasy client;
- its expected use is **one or a few pre-authorized accounts** (a personal NBA
  optimizer, a scheduled import, an internal dashboard);
- tokens are supplied by the operator; there is no multi-tenant onboarding.

Under that lens, many "Medium/High" recommendations add exported surface,
configuration, and maintenance for little real benefit — the exact
over-engineering a solo maintainer should resist. The purpose of this ADR is to
**record which recommendations we decline and why**, so they are not relitigated
at every review, and to state the conditions that would flip each decision.

Reviews keep arriving; this ADR is the standing answer.

## Decision

**Accept cheap correctness fixes that benefit the read tool; decline the
production-SaaS framework-building.** Concretely:

### Accepted (done, e.g. in v2.2.2–v2.2.7)

Typed errors, bounded reads, retries with `Retry-After`, single-flight
context-aware refresh + a `TokenStore` hook, options constructor with
validation, injectable HTTP/cache/logger, dynamic game-key discovery,
pagination for transactions/draft, `DecodeWarning`, the SDK/app split,
`FromEnv` precedence, `Roster.TeamID`, `SlotUnknown`, URL validation
(incl. query/fragment/user-info rejection), refresh-response validation
(empty `access_token` / non-positive `expires_in`), and direct endpoint-fetcher
tests. These are correctness or low-cost wins with clear value even for one user.

### Declined (out of scope unless a trigger below fires)

| Recommendation | Why declined for this project |
|---|---|
| **`Page[T]` metadata + `ListAll`/iterator helpers** | Callers get slices and a 6-line paging loop (documented). Generic page envelopes + iterators are framework surface for a client that returns small collections to one user. |
| **`TokenPersistencePolicy` (required mode) + detached persistence context** | `TokenStore.Save` is advisory by design. For a single operator, a failed save is visible and re-auth is a manual, rare event — not an availability incident. A required-mode + `context.WithoutCancel` timeout is multi-tenant durability machinery. |
| **Rich `APIError` (method, Yahoo code, message, `RetryAfter`, request ID, headers, `Retryable()`)** | `StatusCode` + `Endpoint` + `Body` already supports `errors.As` branching. The rest is normalization for a gateway/observability stack this project does not have. |
| **Transport-error retries (retry `HTTPDoer.Do` errors)** | Requires careful `net.Error` classification (timeout/temporary vs. TLS/cancel/permanent) whose semantics keep shifting. For a personal tool, a transient network error surfacing to the caller (who retries) is acceptable; the risk of wrongly retrying a non-idempotent-looking failure isn't worth it. |
| **Hard `MaxRetryAfter` cap** | Honoring Yahoo's `Retry-After` is correct behavior. A caller who cannot wait sets a request `context` deadline, which already aborts the backoff sleep. A separate cap is redundant config. |
| **Cache payload versioning / schema-prefixed keys** | A model change makes an old entry a decode-miss, which is now *logged* and simply refetched. Version prefixing solves a rolling-upgrade problem a single-process tool does not have. |
| **Ship canonical SQLite migrations / schema versioning** | The cache table is the consumer's to own (documented). The app's schema lives with the app (`cmd/nba-tool`, tested inline). A shipped migration framework is app infrastructure, not SDK surface. |
| **Live sanitized Yahoo fixture corpus across sports/scoring/keeper/etc.** | Valuable but unbounded, and unattainable without live multi-account credentials. Synthetic per-endpoint fixtures (added in v2.2.7) cover the parse boundaries a solo maintainer can actually keep current. |
| **Full initial OAuth authorization flow (authorize URL, callback, state, PKCE, token loading)** | This is a *read resource client*; onboarding is the application's job (`golang.org/x/oauth2` exists). Bundling it would double the module's scope and security surface. The README now states this explicitly. |
| **Standardize player pagination on `PageOptions`** | `GetLeaguePlayers(..., start, count)` predates `PageOptions` and works. Changing it is a breaking churn for cosmetic consistency; not worth a major bump. |
| **Draining retry bodies / oversize typed error / accept full `2xx`** | Micro-optimizations and contract-tidiness with negligible real impact on Yahoo's `200`-returning GET endpoints for one user. Revisit only if a real symptom appears. |

### Deferred (cheap, but low value — not declined outright)

- **`classifySlot`: non-empty unknown slot → starting.** A sport-aware active-slot
  allowlist is the only real fix and is fragile/high-maintenance. Empty slots are
  already `SlotUnknown`; a hypothetical new Yahoo reserve slot is a rare,
  eyeball-detectable event.
- **`fetchLeagues` `fmt.Sscanf` season parse.** Surfacing it needs a
  `DecodeWarnings` field on `League` (more surface) for a value Yahoo does not
  malform.

## Consequences

### Positive

- The module stays small, boring, and maintainable by one person.
- Reviews can be triaged against this ADR instead of re-argued.
- Every accepted change is a genuine correctness or low-cost win.

### Negative

- The SDK is explicitly **not** turnkey for multi-user unattended SaaS; such a
  consumer must add a wrapper (durable atomic token store, richer error
  normalization, their own fixtures). The README and the reviews say so.

## Revisit triggers

Flip the relevant decision if the project's scope changes:

- **Multi-user / unattended service** → required token persistence + detached
  context + atomic store; richer `APIError`.
- **High-volume ingestion under throttling** → transport retries, `MaxRetryAfter`,
  body draining, cache versioning.
- **A public product** → OAuth onboarding, live fixtures, endpoint-support matrix,
  Yahoo attribution/compliance.
- **Write features requested** → a separate module/scope decision entirely.

Until then, these remain deliberately out of scope.

## References

- [assessment 0001](../assessments/0001-maintainable-architect-v4-assessment.md),
  [0002](../assessments/0002-maintainable-architect-v4-assessment.md),
  [0003](../assessments/0003-maintainable-architect-v4-assessment.md)
- [ADR-0002: SDK/app split](./0002-separate-sdk-from-application.md),
  [ADR-0004: token persistence](./0004-token-persistence.md)

---

**Decision Date**: 2026-07-24
**Participants**: Maintainer; maintainable-architect-v4 reviews
**Outcome**: Accepted — accept cheap correctness fixes; decline the multi-user
SaaS hardening backlog for a solo read-only client, with recorded revisit
triggers.
