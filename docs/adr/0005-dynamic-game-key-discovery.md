# ADR 0005: Dynamic Game-Key Discovery

## Status

Accepted (implemented in v2.1.0)

## Context

Yahoo game keys (e.g. NBA 2024 = `454`) are external data that changes every
season. The SDK ships a static `gameIDMap` in `games.go` covering 2001–2025.
The package functions `GetGameID` / `GetGameKey` look up that map and return an
error for any season not in it — so `GetGameKey("nba", 2026)` **fails today**
(assessment finding m2 / item 8). Every new season requires a hand-edited map
and a release.

Yahoo exposes the authoritative mapping through its `games` resource:

```
GET /fantasy/v2/games;game_codes=nba;seasons=2026?format=json
```

which returns the `game_key`/`game_id` for that sport-season. The maintainable
answer is to discover keys at runtime and keep the static map only as an offline
fast path.

## Decision

**Add a client method `Client.GameKey(ctx, gameCode, season)` that discovers the
game key from Yahoo, with the static map as an offline fast path and results
cached.** The package-level `GetGameID` / `GetGameKey` stay unchanged as the
offline-only lookup.

Resolution order:

1. **Static map** — if the sport-season is in `gameIDMap`, return it with no
   request. These are known-good and cover 2001–2025.
2. **Yahoo discovery** — otherwise GET the `games` resource, parse the
   `game_key`, and (if caching is enabled) cache it. Game keys are stable for a
   season, so the cache TTL is long.

The response parser is defensive about Yahoo's irregular JSON: the `games`
collection is an object with numbered string keys plus `count`, and each
`game` may be an array or a single object.

### Why static-first rather than discovery-first

The assessment cautioned against hardcoding as the *only* mechanism. Static-first
keeps discovery as the fallback while avoiding a network round-trip (and rate-limit
exposure) for the 25 known-good seasons. New/unknown seasons — the actual gap —
go to discovery. This is strictly better than either pure approach and keeps the
offline path working with no credentials.

## Consequences

### Positive

- `GameKey(ctx, "nba", 2026)` works without a library release.
- No request for seasons already in the static map; discovery only for the gap.
- Offline callers keep using `GetGameKey` unchanged.
- Cached, so repeated lookups cost one request per season.

### Negative

- Discovery requires a configured client (tokens) and a network call; the
  package functions do not.
- The parser depends on Yahoo's `games` JSON shape, which is irregular and not
  verified here against live traffic (covered by a fixture test instead).

### Mitigation

- Keep the static map as the primary path so the common case needs no network.
- Test the parser against a representative fixture.
- Long cache TTL to minimize requests.

## Alternatives Considered

1. **Extend the static map to 2026 by hand.** Rejected as the *primary* fix — it
   just moves the staleness one year and needs the real IDs each season. (Still
   fine as a stopgap if someone has the IDs.)
2. **Discovery-first, static fallback.** Rejected — a network call for every
   game-key lookup, including 25 known-good seasons, for no benefit.
3. **Auto-populate the static map from discovery at startup.** Over-engineered;
   per-lookup caching achieves the same with less machinery.

## Implementation Details

- `Client.GameKey(ctx, gameCode, season) (string, error)` in `games.go`.
- `games;game_codes=<code>;seasons=<season>` endpoint via the existing
  `makeRequest` (so it inherits retries, typed errors, refresh).
- Parser handles the numbered-key `games` object and array-or-object `game`.
- Cache key `gamekey:<code>:<season>`, TTL 30 days.
- No change to `GetGameID` / `GetGameKey`.

## Success Metrics

- `GameKey(ctx, "nba", 2025)` returns `466` with no HTTP request (static path).
- `GameKey(ctx, "nba", 2026)` performs one request and returns the discovered
  key (verified with a fixture server).
- A second call for the same season is served from cache.

## References

- [assessment 0001](../assessments/0001-maintainable-architect-v4-assessment.md) (finding m2 / item 8)
- Yahoo Fantasy `games` resource: https://developer.yahoo.com/fantasysports/guide/

## Related Decisions

- [ADR-0003](./0003-options-constructor.md) — the client and cache seams this uses.

---

**Decision Date**: 2026-07-24
**Participants**: Maintainer; maintainable-architect-v4 review
**Outcome**: Accepted — add `Client.GameKey` runtime discovery with a
static-map fast path and caching; keep the package functions as the offline path.
