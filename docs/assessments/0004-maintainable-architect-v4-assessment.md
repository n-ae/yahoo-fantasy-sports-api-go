---
assessment: 0004
title: Triage of the fourth external review (v2.2.7) against ADR-0007
date: 2026-07-25
status: Accepted
reviewer: maintainable-architect-v4
scope: pkg/yahoo (SDK) @ v2.2.7, module .../v2, commit be26744
supersedes: none (complements 0001-0003; triages the 4th external review against ADR-0007)
---

# Assessment 0004: yahoo-fantasy-sports-api-go v2.2.7 — review triage

## Executive summary

This is a **triage, not a re-review.** The fourth external review (8.4/10) is
accurate and, by its own admission, mostly re-raises items already settled in
ADR-0007. I verified the three claimed v2.2.7 fixes against code (all real), then
bucketed every finding: **~95% is already declined (ADR-0007), already deferred
(ADR-0007), or already fixed (v2.2.7).** Exactly **one** item is not yet recorded
anywhere as a decision — auth-combination validation — and it is borderline. The
project has reached genuine done-ness for its stated scope; reviews are now
re-litigating a settled ADR.

Verdict: **done.** Optionally close the one open question (auth-combo) with a
6-line guard folded into the next incidental release, or append it to ADR-0007.
Either way it should leave the "unrecorded" state so review #5 can be triaged in
one pass.

---

## 1. Verification of the v2.2.7 fixes (all CONFIRMED)

| Claimed fix | Evidence (file:line) | Verdict |
|---|---|---|
| URL query/fragment/userinfo rejection | `options.go:32-40` — `validateHTTPURL` rejects `RawQuery`/`ForceQuery`, `Fragment`, and `User != nil` after the scheme+host checks | **CONFIRMED** |
| Refresh-response validation (empty `access_token`, non-positive `expires_in`) | `client.go:326-331` — `doRefresh` returns an error before installing creds if `TrimSpace(AccessToken)==""` or `ExpiresIn <= 0` | **CONFIRMED** |
| New endpoint fixture tests raising coverage | `endpoints_test.go` (10 fixture tests: UserLeagues/Teams/Players/PlayerStats/Standings/Matchups/DraftResults/Transactions + empty + malformed); `go test ./pkg/yahoo/` reports **coverage: 78.6%** | **CONFIRMED** |
| README "feature parity" removed / ADR-0007 scope honesty | Recorded as accepted in ADR-0007:44; review agrees | **CONFIRMED** |

The v2.2.7 cycle closed exactly the three cheap wins that assessment 0003 said to
do, plus the README honesty edit. Nothing was faked.

---

## 2. Bucketing every finding in the external review

Buckets: **(a)** already DECLINED in ADR-0007 · **(b)** already DEFERRED in
ADR-0007 · **(c)** already FIXED in v2.2.7 · **(d)** genuinely new / unrecorded.

| # | Review finding | Bucket | Citation |
|---|---|---|---|
| 1 | URL query/fragment/userinfo rejection | **(c)** | `options.go:32-40`; ADR-0007:44 (accepted) |
| 2 | Refresh empty-token / non-positive-expiry rejection | **(c)** | `client.go:326-331`; ADR-0007:44 |
| 3 | Endpoint fixture tests (64.4%→78.6%) | **(c)** | `endpoints_test.go`; measured 78.6% |
| 4 | README "feature parity" removed | **(c)** | ADR-0007:44 |
| 5 | ADR-0007 scope honesty | **(c)** | ADR-0007 whole |
| 6 | **Auth-combination validation weak** (`WithCredentials("k","")`, `WithTokens("","refresh")`) | **(d)** | `options.go:280-288` — no such check exists; not in ADR-0007 (see §3) |
| 7 | Token persistence advisory + request ctx | **(a)** | ADR-0007:52 (`TokenPersistencePolicy` + detached context row) |
| 8 | Transport errors not retried | **(a)** | ADR-0007:54 (Transport-error retries row) |
| 9 | Retry-After exceeds MaxBackoff | **(a)** | ADR-0007:55 (Hard `MaxRetryAfter` cap row) |
| 10 | Retry bodies not drained | **(a)** | ADR-0007:61 (Draining retry bodies row) |
| 11 | Only 200 accepted (204→APIError) | **(a)** | ADR-0007:61 (accept full `2xx` row) |
| 12 | Oversize response truncated at 10MiB, no `ErrResponseTooLarge` | **(a)** | ADR-0007:61 (oversize typed error row) |
| 13 | APIError body ~10MiB / thin APIError | **(a)** | ADR-0007:53 (Rich `APIError` row) |
| 14 | Unknown non-empty roster slot ("RES") → starting | **(b)** | ADR-0007:65-68 (deferred: `classifySlot`) |
| 15 | League-season `fmt.Sscanf` silent | **(b)** | ADR-0007:69-71 (deferred: `fetchLeagues` `Sscanf`) |
| 16 | GameKey negative season → network; `parseGameKey` returns first entry | **(d)-trivial** | Not in ADR-0007; recorded 0003 #13/#14 as "defensible as-is"; static map covers 2001-2025 so path rarely reached |
| 17 | Pagination no metadata / iterator / `ListAll` | **(a)** | ADR-0007:51 (`Page[T]` + `ListAll`/iterator row) |
| 18 | Player pagination inconsistent (raw start,count) | **(a)** | ADR-0007:60 (standardize player pagination row) |
| 19 | Cache payload not versioned | **(a)** | ADR-0007:56 (cache payload versioning row) |
| 20 | Expired-row cleanup errors ignored | **(a)-spirit** | Covered by ADR-0007 "cache is app concern" (56/57); 0003 #22 = Trivial |
| 21 | SQLite migration caller-owned | **(a)** | ADR-0007:57 (ship canonical migrations row) |
| 22 | OAuth onboarding absent | **(a)** | ADR-0007:59 (full OAuth onboarding row) |
| 23 | Writes absent | n/a scope | Documented read-only scope; not a defect (ADR-0007 context, 0003 #24) |
| 24 | Live fixture corpus missing | **(a)** | ADR-0007:58 (live fixture corpus row) |
| 25 | GetTeamRoster wrapper 0% cov; APIError.Error/DecodeWarning.String/noopLogger.Printf 0% | **(d)-trivial** | See §3; `fetchRoster` is 89.5% covered |
| 26 | Ecosystem maturity / release velocity | n/a strategic | Not code; mitigation ("pin it, wrap it") is already the operating model (0003 #26) |

**Tally:** 5× (c) fixed, 12× (a) declined, 2× (b) deferred, 2 scope/strategic
non-defects, and **2× (d)** — of which one (GameKey/coverage triviality) 0003
already assessed as defensible-as-is. That leaves **auth-combo validation as the
single live, unsettled item.** The review's own annotations concede this; it adds
essentially nothing beyond confirming v2.2.7 and re-listing ADR-0007.

---

## 3. The one live candidate — auth-combination validation (finding #6)

### Is there any auth-combo validation today? No.

`NewClient` (`options.go:280-288`) validates only: non-nil `httpClient`, cache
present when `cacheEnabled`, and non-nil `logger`. `WithCredentials`
(`options.go:127-133`) and `WithTokens` (`options.go:136-142`) do zero
cross-field validation. So today:

- `WithCredentials("key","")` (half a credential pair) → accepted; refresh later
  fails with a Yahoo `401` because the Basic auth header is malformed.
- `WithTokens("","refresh-only")` with no creds → accepted; the refresh token can
  never be exchanged (no key/secret), and the first request fails the
  `token == ""` guard in `makeRequest` (`client.go:348-350`).

Failure is deferred to the first request in both cases.

### Would the ~6-line guard be correct? Yes, and it rejects only impossible configs.

The valid-config matrix has legitimate partial forms the guard must NOT reject:
- creds + refresh, no access token → valid (refresh-first).
- access token only, no creds/refresh → valid (read until expiry).
- creds + access + refresh → full setup.

Only two combinations can **never** work: `(key=="") != (secret=="")` (you can
never authenticate with half a pair) and a refresh token with no creds (nothing
to exchange it with). A guard rejecting exactly those is unambiguously correct and
excludes no valid setup.

### Recommendation: cheap-but-borderline — close it, don't ceremony it.

This is the one finding that is genuinely a fail-fast correctness improvement
rather than SaaS gold-plating: pure addition, no exported surface, no breaking
change, ~6 lines + one table test. It fits ADR-0007's own **"accept cheap
correctness fixes"** principle (ADR-0007:33).

But its practical value for a solo operator is **low**: the failure is already
caught loudly on the first request, within seconds of a one-time wiring step. It
converts a clear first-request error into a clear construction error — a real but
small ergonomic gain.

**So:**
- **Do NOT cut a dedicated v2.2.8 for this alone.** A version bump + changelog +
  tag for a fail-fast guard on impossible config is ceremony exceeding the value.
- **Do** fold the guard into `NewClient` on the next incidental touch of
  `options.go` (add a `TestAuthComboValidation` table alongside
  `options_test.go`), OR — if you would rather not touch code — **append it to
  ADR-0007** so review #5 triages it in the DECLINED/DEFERRED table instead of
  re-raising it as "new."

Writing the 6-line ADR paragraph costs about what the 6-line guard + test costs;
given it matches ADR-0007's accepted bucket, I lean slightly toward **implementing
it** and closing the question permanently. Either path is defensible. The only
wrong move is leaving it unrecorded for a fifth review to "discover."

### GetTeamRoster 0% coverage — trivial, ignore.

`GetTeamRoster` (`client.go:241`) is a thin cache-key + cacheGet/cacheSet wrapper
around `fetchRoster` (89.5% covered). The 0% is the cache plumbing, exercised
everywhere else. Same for `APIError.Error`/`DecodeWarning.String`/
`noopLogger.Printf`. No action; not worth a test that asserts a `Sprintf`.

---

## 4. Maintainability verdict (solo lens) + C4 note

**Architecture unchanged since 0003 — no redraw needed.** The C4 context and
container diagrams in `docs/assessments/0003-...md` (§C4 context / §C4 container,
lines 108-158) remain accurate: one Go module, one `pkg/yahoo.Client`, optional
caller-owned SQLite cache, advisory `TokenStore`, Yahoo REST + OAuth2 refresh. The
two request-path hazards that diagram annotated (base-URL swallow, unvalidated
refresh) are both **now fixed in v2.2.7**, so the flowchart's two danger callouts
are closed. Nothing in this review moves a box or an edge.

**Is there anything here worth doing, or is the project at genuine done-ness?**

I agree with your read, plainly: **the project is at done-ness for its declared
scope.** Reviews have entered the phase of re-litigating ADR-0007 — which is
exactly what that ADR exists to short-circuit, and it is doing its job. This
fourth review, by volume, is ~95% (a)/(b)/(c). The single new-and-actionable item
is **auth-combo validation, and it is borderline** — a correct, cheap fail-fast
guard whose real-world payoff is small because the failure already surfaces at
first use.

Everything else the review lists is a deliberate boundary, and the review says so
itself. The honest state: this is a boring, well-factored, 78.6%-covered,
single-binary read-only SDK that a solo maintainer can own for years. Ship it as
the dependency it is. The next review should be triaged against ADR-0007 in one
pass, and — if you take the recommendation — auth-combo should already be closed
in it.

**Scores stay where 0003 put them:** ~9/10 as a pinned personal read-only
dependency; the review's 8.4 (prod) / 6.5 (turnkey-SaaS) are just the multi-user
lens ADR-0007 already declined to optimize for.

---

## Documentation action (optional, one line)

If you implement the auth-combo guard: note it in the CHANGELOG (behavioral
tightening of `NewClient`, not an API addition) and add one line to ADR-0007's
**Accepted** paragraph. If you decline it: add one row to ADR-0007's **Deferred**
section ("auth-combo validation — cheap/correct but failure already surfaces at
first request; low value for a single pre-wired operator"). No new ADR; no
diagram change.
