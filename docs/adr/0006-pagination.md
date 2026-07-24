# ADR 0006: Pagination for Collection Endpoints

## Status

Accepted (implemented in v2.2.0)

## Context

`GetLeagueTransactions` and `GetLeagueDraftResults` exposed no pagination
controls, so their completeness depended entirely on Yahoo's default page size
(assessment finding m4 / item 9). `GetLeaguePlayers` already took `start, count`,
so the client was inconsistent. Transaction history in particular can exceed one
default page.

## Decision

**Add a `PageOptions{Start, Count}` type and paged method variants**
(`GetLeagueTransactionsPage`, `GetLeagueDraftResultsPage`) that append Yahoo's
`;start=N;count=M` segment. The existing no-argument methods are kept and
delegate to the paged variants with the zero `PageOptions` (Yahoo default),
preserving behavior.

The zero value means "no explicit pagination"; a `Count <= 0` uses Yahoo's
default page size. Cache keys include start/count so pages don't collide.

### Why not iterators or a ListAll helper

An `iter.Seq` iterator or a `ListAllTransactions` that auto-pages until
exhaustion would be convenient, but both depend on knowing when Yahoo signals
the last page (a short page), which is behavior this SDK cannot verify against
live traffic here. A single-page method with explicit `PageOptions` is
well-defined, testable, and lets callers write the trivial paging loop
themselves. An iterator can be layered on later without a breaking change if a
clear need appears.

## Consequences

### Positive

- Transactions and draft results are now fully retrievable via explicit paging.
- Consistent with `GetLeaguePlayers`' existing `start, count`.
- Backward-compatible: the no-arg methods are unchanged.

### Negative

- Callers who want "everything" write a small loop (documented in the README).
- Two methods per endpoint (default + paged) — mild surface growth.

## Alternatives Considered

1. **Change the existing methods to take `start, count`.** Rejected — breaking,
   and not needed since additive variants suffice.
2. **Range-over-func iterator.** Deferred — depends on unverified end-of-pages
   behavior; can be added non-breakingly later.
3. **`ListAllTransactions` convenience.** Deferred for the same reason.

## Implementation Details

- `PageOptions` and its `suffix()` live in `pagination.go`.
- `fetchTransactions` / `fetchDraftResults` take `PageOptions` and append the
  suffix; the paged public methods and the default delegates wrap them.
- Cache keys: `league:<key>:transactions:<start>:<count>` (and draft likewise).

## References

- [assessment 0001](../assessments/0001-maintainable-architect-v4-assessment.md) (finding m4 / item 9)
- Yahoo Fantasy resource paging: https://developer.yahoo.com/fantasysports/guide/

---

**Decision Date**: 2026-07-24
**Participants**: Maintainer; maintainable-architect-v4 review
**Outcome**: Accepted — add `PageOptions` and paged method variants for
transactions and draft results; keep the no-arg methods as zero-page delegates;
defer iterators/ListAll.
