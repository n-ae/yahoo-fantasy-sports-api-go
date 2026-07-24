// Package service contains the opinionated NBA analytics application layer
// (league import, valuation, trade analysis) built on top of the reusable
// pkg/yahoo SDK.
//
// Deprecated: this package is application code, not part of the reusable SDK.
// It is scheduled to move under cmd/nba-tool and be removed from the module's
// public surface in v2. See docs/adr/0002-separate-sdk-from-application.md and
// docs/v2-roadmap.md. Depend on pkg/yahoo directly and own your own analytics.
package service
