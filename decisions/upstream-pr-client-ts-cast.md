> **Submitted**: 2026-04-12 · PR: https://github.com/Wei-Shaw/sub2api/pull/1591

# Upstream PR Draft: Strict Cast in API Client

**Target**: https://github.com/Wei-Shaw/sub2api
**Source Branch**: `roswellcsy/sub2api:fix/client-ts-strict-cast`
**Base**: `upstream/main`

---

## Title

fix(frontend): use double cast for ApiResponse → Record to satisfy vue-tsc -b

## Summary

Fix `TS2352` error in `frontend/src/api/client.ts:95` that blocks `vue-tsc -b` strict build. Production `pnpm run build` currently fails with exit code 2, so `dist/` cannot be produced.

## Root cause

Single-step type assertion `apiResponse as Record<string, unknown>` violates TypeScript's stricter rules under project references (`vue-tsc -b`). `ApiResponse<unknown>` lacks an index signature, so the conversion is flagged as potentially unsafe:

```
src/api/client.ts:95:22 - error TS2352: Conversion of type 'ApiResponse<unknown>' to type 'Record<string, unknown>' may be a mistake because neither type sufficiently overlaps with the other. If this was intentional, convert the expression to 'unknown' first.
```

Note: `pnpm run typecheck` (which runs `vue-tsc --noEmit`) does **not** catch this under looser single-project compilation semantics on some environments, but the error surfaces reliably under `-b` incremental project-references mode used by the production `build` script.

## Fix

Use double cast `as unknown as Record<string, unknown>` to explicitly acknowledge the intentional type widening. Equivalent semantics, zero runtime impact — only appeases the stricter type checker.

```ts
// before
const resp = apiResponse as Record<string, unknown>

// after
const resp = apiResponse as unknown as Record<string, unknown>
```

## Test plan

- [ ] `pnpm run typecheck` passes
- [ ] `pnpm run build` passes (currently fails on this line)
- [ ] `dist/` produces expected bundle
- [ ] Runtime behavior unchanged (error path still surfaces `reason` / `metadata` from upstream wire fix `faee59ee`)

## How to open this PR (for repo owner)

```bash
cd /path/to/sub2api-fork
git checkout -b fix/client-ts-strict-cast upstream/main
# cherry-pick or manually apply the 1-line change from main
git push origin fix/client-ts-strict-cast
gh pr create --repo Wei-Shaw/sub2api --base main --head roswellcsy:fix/client-ts-strict-cast \
  --title "fix(frontend): use double cast for ApiResponse → Record to satisfy vue-tsc -b" \
  --body-file decisions/upstream-pr-client-ts-cast.md
```
