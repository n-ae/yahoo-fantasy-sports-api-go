# Migrating to v2

v2 is a breaking release. It finalizes the options constructor, modernizes the
cache interface, and removes the bundled NBA application from the library's
public surface. See [docs/v2-roadmap.md](./v2-roadmap.md) for the why.

## 1. Import path

v2 modules use a `/v2` suffix:

```go
// v1
import "github.com/n-ae/yahoo-fantasy-sports-api-go/pkg/yahoo"

// v2
import "github.com/n-ae/yahoo-fantasy-sports-api-go/v2/pkg/yahoo"
```

```bash
go get github.com/n-ae/yahoo-fantasy-sports-api-go/v2@v2.0.0
```

The `v1.x` line remains available and maintained on the `v1-maintenance` branch.

## 2. Constructor

The positional `NewClient(apiKey, apiSecret, db)` is removed. `NewClient` is now
the validating, functional-options constructor (formerly `NewClientWithOptions`):

```go
// v1
client := yahoo.NewClient("", "", db)

// v2
client, err := yahoo.NewClient(
    yahoo.WithCredentials(consumerKey, consumerSecret),
    yahoo.WithTokens(accessToken, refreshToken),
    yahoo.WithSQLiteCache(db), // optional
    yahoo.FromEnv(),           // optional: fill gaps from YAHOO_* env vars
)
if err != nil {
    return err
}
```

If you relied on the old implicit environment loading, add `yahoo.FromEnv()`
explicitly.

## 3. Cache interface

The `Cache` interface is now context-aware and `[]byte`-based:

```go
// v1
type Cache interface {
    Get(key string) (string, error)
    Set(key string, value interface{}, ttl time.Duration) error
    Delete(key string) error
}

// v2
type Cache interface {
    Get(ctx context.Context, key string) (value []byte, ok bool, err error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}
```

A miss is now `ok == false` (not an error). The bundled SQLite cache is unchanged
in behavior; enable it with `WithSQLiteCache(db)` as before. The `APICache` type
is no longer exported — interact via the `Cache` interface.

## 4. Roster model

`Roster.Position` (the deprecated single primary position) is removed. Use
`EligiblePositions []string` (and `SlotState` / `IsStarting`) instead:

```go
// v1
pos := entry.Position

// v2
var pos string
if len(entry.EligiblePositions) > 0 {
    pos = entry.EligiblePositions[0]
}
```

## 5. Removed: pkg/service and pkg/repository

The NBA analytics application (`pkg/service`, `pkg/repository`) is no longer part
of the module's public API. It now lives under `cmd/nba-tool/internal/` and is
not importable. If you depended on it, either pin `v1.x`, vendor the code, or
build your own analytics on top of `pkg/yahoo` (recommended).

## Summary

| v1 | v2 |
|---|---|
| `.../pkg/yahoo` | `.../v2/pkg/yahoo` |
| `NewClient(key, secret, db)` | `NewClient(WithCredentials(...), WithTokens(...), WithSQLiteCache(db))` |
| `Cache.Get(key) (string, error)` | `Cache.Get(ctx, key) ([]byte, bool, error)` |
| exported `APICache` | internal; use `WithSQLiteCache` / the `Cache` interface |
| `Roster.Position` | `Roster.EligiblePositions` |
| import `.../pkg/service` | moved to `cmd/nba-tool` (not a library API) |
