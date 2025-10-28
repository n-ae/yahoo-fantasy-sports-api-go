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

## Future ADRs

Planned topics for future architectural decisions:

- **ADR-0002**: Sport-specific stat definitions and mappings (NFL, MLB, NHL)
- **ADR-0003**: Caching strategy for stat metadata and league settings
- **ADR-0004**: Error handling patterns across API methods
- **ADR-0005**: API rate limiting and retry strategies
- **ADR-0006**: CLI tool architecture for data export

## Contributing

When making significant architectural decisions:

1. Copy `template.md` to a new file with the next number
2. Fill in all sections thoroughly
3. Discuss with team before marking as "Accepted"
4. Update this README index
5. Link related code to the ADR in comments

## Template

See [template.md](./template.md) for the ADR template.
