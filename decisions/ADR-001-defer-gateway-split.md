# ADR-001: Defer `gateway_service.go` Structural Split

- Status: Accepted (2026-04-12)

## Context

API Station (`sub2api` fork) Phase 1-3 is complete. The fork-specific runtime logic has already been physically aggregated into `backend/internal/pkg/apistation/`, including:

- `fingerprint.go`
- `cooldown.go`
- `request_audit.go`
- `signature_cache.go`
- `sensitive_words.go`
- `version_drift.go`
- `account_events.go`

The corresponding `apistation_*` settings wiring has also been completed across backend settings views, admin DTOs, and frontend settings controls, so the current question is whether to proceed with the originally proposed Phase 4 structural split of `backend/internal/service/gateway_service.go`.

The quantitative evidence does not support doing that split now:

- `gateway_service.go` is 8,900 lines with 31 functions.
- `Forward()` is the largest function at 594 lines.
- The fork's cumulative intrusion into `gateway_service.go` is only `+50/-9 = 59` lines across Phase 1-3.
- The API Station hook surface inside `Forward()` is only 2 points:
  - line 4074: thinking block signature cache
  - line 4476: request audit
- New API Station behavior is already isolated in `internal/pkg/apistation/`, rather than spread across multiple gateway methods.
- Upstream changed `gateway_service.go` very frequently over the last 6 months: 267 commits, `+14443/-5339`.
- `GatewayService` currently has 25 constructor dependencies and is called directly by 5 or more handlers.
- The beta header handling direction is converging with upstream, including upstream commits `e51c9e50` and `7c60ee3c`, which makes upstream contribution more realistic than a long-lived fork-only structural divergence.

From those facts:

1. Structural splitting does not remove merge pressure when the actual conflict hotspot is the same logical function (`Forward()`), not the physical file boundary.
2. The fork's changes are already physically aggregated where it matters: new logic lives in `internal/pkg/apistation/`.
3. The remaining gateway edits are narrow integration hooks that are plausible upstream PR candidates.
4. The split itself is high risk because of the 25-constructor dependency surface and the existing coupling inside `Forward()`.

## Decision

Phase 4 structural splitting of `backend/internal/service/gateway_service.go` is deferred.

Instead, the project adopts a three-part strategy:

1. Monthly upstream merge practice to measure real conflict pressure.
2. Stronger API Station markers at the narrow integration points in `gateway_service.go`.
3. Upstream PRs for changes that are broadly useful, especially beta-header related convergence points.

This keeps the fork close to upstream until measured evidence shows that a structural split is necessary.

## Consequences

### Positive

- Avoids introducing a high-risk refactor into a service with 25 constructor dependencies and wide handler reach.
- Keeps file layout and service boundaries aligned with upstream, which reduces future cherry-pick and merge bookkeeping cost.
- Preserves the current pattern where fork-specific logic is isolated in `internal/pkg/apistation/` and gateway changes remain narrow adapter hooks.
- Favors upstreaming small, convergent changes instead of deepening long-lived fork-only structure.

### Negative

- `gateway_service.go` will continue to grow from its current 8,900 lines.
- `Forward()` remains 594 lines and is still difficult to read in one pass.
- Each upstream merge still requires explicit human review of the API Station hook points in `Forward()`.

## Alternatives Considered

### Option A: 3-service facade split

Rejected.

Reason: `GatewayService` currently has 25 constructor dependencies and is directly called by 5 or more handlers. A facade split would force constructor reshaping, handler rewiring, and responsibility redistribution across a broad surface area before it delivers any proven merge benefit.

### Option B: Progressive internal module extraction

Rejected for now.

Reason: extracting helper modules from inside `gateway_service.go` improves local readability, but it does not materially reduce merge conflicts when the upstream churn and fork hooks still collide in the same logical decision points inside `Forward()`.

### Option C: Monthly upstream merge practice + stronger markers + upstream PRs

Accepted.

Reason: this option measures actual conflict cost before paying refactor risk, keeps the fork structurally close to upstream, and concentrates work on the two existing integration hooks plus isolated package logic.

## Re-evaluation Triggers

Re-open the Phase 4 split discussion if any of the following happens:

- A single upstream merge produces more than 100 conflict lines or requires more than 1 hour of manual resolution.
- Upstream performs a structural split of `Forward()` or otherwise decomposes `gateway_service.go`.
- The fork's cumulative change in `gateway_service.go` exceeds 500 lines.

## Appendix

Current API Station markers in `backend/internal/service/gateway_service.go`:

1. Line 4074: `// api-station: begin - thinking block signature cache (from CLIProxyAPI)`
2. Line 4476: `// api-station: request audit`
