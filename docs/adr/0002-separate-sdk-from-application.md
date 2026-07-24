# ADR 0002: Separate the Reusable SDK from the Bundled Application

## Status

Proposed

## Context

The module `github.com/n-ae/yahoo-fantasy-sports-api-go` currently ships two
very different kinds of code in one importable unit:

- **`pkg/yahoo`** — a reusable, read-only Yahoo Fantasy Sports client
  (transport, OAuth refresh, response conversion, optional caching). This is the
  code most consumers actually want.
- **`pkg/service` + `pkg/repository`** — an opinionated NBA analytics
  *application*: league import, trade evaluation, player valuation, and a
  concrete SQLite schema, complete with hardcoded scoring weights and a fixed
  `2024-25` season.

Because both live under one module, anyone importing the client inherits the
whole thing, including:

- a **mandatory cgo dependency** (`github.com/mattn/go-sqlite3`), which forces
  `CGO_ENABLED=1`, a C toolchain, and slower cross-compilation on every
  consumer — even those who never touch caching or persistence;
- application logic (scoring models, valuation math, a private DB schema) that
  is not configuration and is wrong for any league that isn't the author's;
- an identity that is neither cleanly "a library" nor cleanly "an app."

The v4 maintainability assessment
([docs/assessments/0001](../assessments/0001-maintainable-architect-v4-assessment.md))
identifies this SDK/application conflation as **the root cause** (finding C1)
behind several other findings: the god-object constructor, the DB-required
`NewClient`, the concrete-SQLite cache, and the application-layer correctness
bugs. Almost every downstream problem gets smaller once the boundary exists.

This is a solo-maintained project. The goal is long-term velocity and minimal
operational surface, not a large modular framework. The decision must therefore
buy real simplification, not add ceremony.

## Decision

**Draw a hard boundary: `pkg/yahoo` is the SDK; the service/repository code is
an application that depends on the SDK, not the other way around.**

Concretely, in order of preference:

1. **Move `pkg/service` and `pkg/repository` out of the library's public
   surface.** Preferred target is a `cmd/`-based application (e.g.
   `cmd/nba-tool/`) or a separate repository/module. The SDK keeps zero
   knowledge of scoring, valuation, or any application schema.
2. **Remove the mandatory cgo SQLite dependency from `pkg/yahoo`.** Caching
   becomes an injected interface with an in-memory default (see
   [ADR-0003](./0003-options-constructor.md) and the planned `Cache`
   interface). The SQLite-backed cache, if kept, moves to the application side.
3. **Depend downward only.** The application imports `pkg/yahoo`; `pkg/yahoo`
   imports nothing from the application. A `go mod graph` / import audit
   enforces this.

The SDK's public promise becomes narrow and honest: "a typed, read-only Yahoo
Fantasy client." Everything opinionated (scoring, season, valuation, storage)
belongs to the consumer.

## Consequences

### Positive

- **No forced cgo for library users.** Importers of `pkg/yahoo` get a
  pure-Go dependency graph and fast cross-compilation.
- **Smaller, cohesive public API.** The SDK is judged on its read path, which
  is its strongest part, instead of on application code that ships as
  importable-but-unsupported.
- **Application bugs stop being SDK bugs.** Hardcoded scoring, the fixed
  `2024-25` season, and valuation edge cases (see assessment finding M5) are no
  longer part of the library's contract.
- **Clearer versioning.** SemVer promises apply to a surface small enough to
  keep stable (reinforces [ADR-0003](./0003-options-constructor.md) and the
  README versioning policy).

### Negative

- **Breaking change for anyone importing `pkg/service` / `pkg/repository`.**
  The import path for that code changes or the code leaves the module.
- **Two things to release** if a second module is used (SDK + app), each with
  its own tags.
- **Short-term churn**: moving packages, updating imports, and re-pointing the
  example programs.

### Mitigation Strategies

- Keep everything in **one repository** and prefer `cmd/` over a second module
  first; only split into a separate module if the app grows its own release
  cadence. This avoids multi-module tagging overhead for a solo maintainer.
- Ship the move in a clearly labeled **major or clearly-documented minor**
  release with a migration note in `CHANGELOG.md` and README.
- Provide a thin migration section: "if you imported `pkg/service`, it now
  lives at `cmd/nba-tool` and is not a supported library API."

## Alternatives Considered

1. **Leave everything in one module, document it as "an app, not a library."**
   - Pros: zero code movement.
   - Cons: still forces cgo on library users; still exports application logic as
     if it were API; does not fix the root cause. Rejected.
2. **Split into two separate modules/repositories immediately.**
   - Pros: cleanest separation; independent release cadence.
   - Cons: multi-module maintenance overhead (two `go.mod`, two tag streams) is
     a poor fit for a solo maintainer until the app justifies it. Reconsider
     once the application has real external users.
3. **Delete the service/repository code entirely.**
   - Pros: maximum simplification of the library.
   - Cons: throws away working, useful sample application code. Preserving it in
     `cmd/` keeps the value without polluting the SDK. Rejected for now.

## Implementation Details

- Target layout (single repo, one module):
  ```
  pkg/yahoo/            # SDK: transport, convert, cache interface — no cgo
  cmd/nba-tool/         # application: service + repository + SQLite schema
  examples/*/main.go    # small SDK usage programs (already independent mains)
  ```
- Enforce direction with an import audit in CI (fail if `pkg/yahoo` imports
  `cmd/...` or any application package).
- Coordinate with [ADR-0003](./0003-options-constructor.md): the SDK's cache
  becomes an injected `Cache` interface, which is what lets SQLite leave the
  core.
- Breaking change: yes, for `pkg/service` / `pkg/repository` importers. The
  `pkg/yahoo` surface stays source-compatible through this move.

## Examples

Before — a CLI must pull in cgo SQLite just to read a roster:

```go
db, _ := sql.Open("sqlite3", "./fantasy.db") // cgo required
client := yahoo.NewClient("", "", db)         // db mandatory
```

After — the SDK has no storage dependency; the app owns its DB:

```go
// library user (pure Go)
client, _ := yahoo.NewClient(yahoo.WithHTTPClient(httpClient))

// application (cmd/nba-tool) owns SQLite and scoring
app := nbatool.New(db, scoringConfig, client)
```

## Success Metrics

- `go list -deps ./pkg/yahoo/...` contains **no** `mattn/go-sqlite3`.
- A consumer can `go build` a program using `pkg/yahoo` with
  `CGO_ENABLED=0`.
- CI import-direction check passes (SDK never imports application packages).
- No further correctness findings filed against `pkg/yahoo` that actually
  originate in application logic.

## References

- [docs/assessments/0001 — Maintainability Assessment](../assessments/0001-maintainable-architect-v4-assessment.md) (finding C1, M5)
- Go modules & cgo: https://pkg.go.dev/cmd/cgo
- Standard Go project layout discussion: https://go.dev/doc/modules/layout

## Related Decisions

- [ADR-0003: Options-Based Constructor with Typed Configuration Errors](./0003-options-constructor.md)
  — the injected `Cache`/`HTTPDoer` seams that make removing SQLite from the
  core possible.

## Notes

- Prefer the `cmd/` approach over a second module until the application has
  external users; revisit if release cadences diverge.
- Open question: whether the SQLite cache implementation is kept as an optional
  sub-package (e.g. `pkg/yahoo/sqlitecache`) that users opt into, so cgo is
  strictly opt-in rather than removed.

---

**Decision Date**: 2026-07-24
**Participants**: Maintainer; maintainable-architect-v4 review
**Outcome**: Proposed — separate the reusable SDK (`pkg/yahoo`) from the bundled
NBA application (`pkg/service`, `pkg/repository`), removing mandatory cgo SQLite
from the library and depending downward only.
