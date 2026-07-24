# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records for the Yahoo Fantasy Sports API Go SDK.

## What is an ADR?

An Architecture Decision Record (ADR) captures an important architectural decision made along with its context and consequences.

## Format

Each ADR follows this structure:
- **Title**: Short descriptive title
- **Status**: Proposed | Accepted | Deprecated | Superseded
- **Context**: What forces are at play (technical, political, social, project)
- **Decision**: What we decided to do
- **Consequences**: What becomes easier or more difficult

## Index

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [0001](./0001-expose-attempt-made-stats.md) | Expose Attempt and Made Stats in SDK | Accepted | 2025-10-28 |
| [0002](./0002-separate-sdk-from-application.md) | Separate the Reusable SDK from the Bundled Application | Proposed | 2026-07-24 |
| [0003](./0003-options-constructor.md) | Options-Based Constructor with Typed Configuration Errors | Proposed | 2026-07-24 |

## Decision Summary

### ADR-0001: Expose Attempt and Made Stats in SDK

**Decision**: Provide three complementary approaches for accessing player statistics:

1. **Sport-Specific Parsed Structs** (Primary - Best DX)
   - `ParseNBAStats()` returns typed `NBAStats` struct
   - Fields: `ThreePointsAttempt`, `ThreePointsMade`, `FGA`, `FGM`, etc.
   - ✅ Use for: Standard leagues, best developer experience

2. **Stat Helper with Constants** (Fallback - Flexible)
   - `StatHelper.GetIntByID(yahoo.StatID3PA)`
   - Named constants for all common stats
   - ✅ Use for: Custom leagues, specific stat queries

3. **Raw Stat Array Access** (Power Users - Maximum Flexibility)
   - Direct access to `[]Stat` array
   - Manual iteration and parsing
   - ✅ Use for: Debugging, custom stats, edge cases

**Rationale**: Graduated API surface allows users to start simple and graduate to advanced usage as needed. Preserves backward compatibility while improving developer experience.

### ADR-0002: Separate the Reusable SDK from the Bundled Application

**Decision** (Proposed): Draw a hard boundary between `pkg/yahoo` (the reusable, read-only SDK) and `pkg/service` + `pkg/repository` (an opinionated NBA application). Move the application code to `cmd/`, drop the declared SQLite requirement from the SDK module (graph hygiene — `pkg/yahoo`-only consumers already build cgo-free), and depend downward only. Addresses assessment finding C1.

### ADR-0003: Options-Based Constructor with Typed Configuration Errors

**Decision** (Proposed): Replace `NewClient(apiKey, apiSecret string, db *sql.DB) *Client` with `NewClient(opts ...Option) (*Client, error)`. Validate configuration at construction, make the database optional, move environment loading behind an explicit `FromEnv()`, and inject HTTP/cache/logger dependencies. Addresses assessment findings M2, M3.

## Future ADRs

Planned topics for future architectural decisions:

- **ADR-0004**: Replace manual OAuth refresh with `oauth2.TokenSource` (single-flight + token persistence)
- **ADR-0005**: HTTP resilience — rate-limit (429/Retry-After) and bounded 5xx retry strategy
- **ADR-0006**: Dynamic game-key discovery vs. the static `games.go` map
- **ADR-0007**: Sport-specific stat definitions and mappings (NFL, MLB, NHL)
- **ADR-0008**: CLI tool architecture for data export

## Contributing

When making significant architectural decisions:

1. Copy `template.md` to a new file with the next number
2. Fill in all sections thoroughly
3. Discuss with team before marking as "Accepted"
4. Update this README index
5. Link related code to the ADR in comments

## Template

See [template.md](./template.md) for the ADR template.
