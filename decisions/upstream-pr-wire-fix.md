> **Submitted**: 2026-04-12 · PR: https://github.com/Wei-Shaw/sub2api/pull/1588

# Upstream PR Draft: Wire Fix

**Target**: https://github.com/Wei-Shaw/sub2api
**Source Branch**: `roswellcsy/sub2api:fix/wire-check-regeneration`
**Base**: `upstream/main` (97f14b7a)

---

## Title

fix(wire): restore wire check/generate with variadic wrapper + ProviderSet

## Body

### Summary

`wire check` currently fails on `upstream/main`, which prevents `go generate` from regenerating `cmd/server/wire_gen.go` cleanly. This patch fixes the wire configuration without changing runtime behavior.

### Root causes

Three distinct problems in the current DI graph:

1. **Variadic provider**: `NewOAuthRefreshAPI(accountRepo, tokenCache, lockTTL ...time.Duration)` uses a variadic tail. Wire cannot inject variadic slices, so it reports `no provider found for []time.Duration`.
2. **Missing interface binding**: `cmd/server/wire.go` lists `payment.ProvideRegistry`, `payment.ProvideEncryptionKey`, and `payment.ProvideDefaultLoadBalancer` individually, which bypasses `payment.ProviderSet`. `ProviderSet` carries the `wire.Bind(new(LoadBalancer), new(*DefaultLoadBalancer))` that `PaymentService` requires, so `LoadBalancer` ends up with no provider.
3. **Duplicate bindings**: `service.ProvidePaymentConfigService` and `service.ProvidePaymentOrderExpiryService` are each bound twice — once through `service.ProviderSet` and once via explicit entries in `cmd/server/wire.go`. Wire reports `multiple bindings` for both types and aborts before reporting problems 1 and 2.

### Changes

- `internal/service/wire.go`: Add `ProvideOAuthRefreshAPI` fixed-arity wrapper. Swap `NewOAuthRefreshAPI` → `ProvideOAuthRefreshAPI` in `ProviderSet`.
- `cmd/server/wire.go`: Replace the three individual `payment.*` providers with `payment.ProviderSet`. Drop the duplicate `service.ProvidePaymentConfigService` and `service.ProvidePaymentOrderExpiryService` entries (already included in `service.ProviderSet`).
- `cmd/server/wire_gen.go`: Regenerated from the corrected config.

### Test plan

- [x] `cd backend/cmd/server && go run github.com/google/wire/cmd/wire check` passes with no errors
- [x] `cd backend/cmd/server && go run github.com/google/wire/cmd/wire` regenerates `wire_gen.go` with no manual patching
- [x] `cd backend && go build ./...` passes
- [ ] `go test ./internal/service/... ./internal/handler/... ./internal/payment/...` (recommended for upstream maintainer)

### How to open this PR (for repo owner)

```bash
cd /Users/chensiyuan/Projects/NekoClaw/workspace/api-station/sub2api-fork
gh pr create \
  --repo Wei-Shaw/sub2api \
  --base main \
  --head roswellcsy:fix/wire-check-regeneration \
  --title "fix(wire): restore wire check/generate with variadic wrapper + ProviderSet" \
  --body-file decisions/upstream-pr-wire-fix.md
```
