#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

if ! git -C "${repo_root}" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Error: ${repo_root} is not a git repository." >&2
  exit 1
fi

if [[ "$(pwd -P)" != "${repo_root}" ]]; then
  echo "Error: run this script from the sub2api-fork root:" >&2
  echo "  cd ${repo_root}" >&2
  exit 1
fi

fetch_status="ok"
fetch_note="Fetched latest refs from upstream."
if ! fetch_output="$(git fetch upstream 2>&1)"; then
  fetch_status="warning"
  fetch_note="git fetch upstream failed; using cached upstream refs."
  fetch_output="${fetch_output//$'\n'/ }"
fi

if ! git rev-parse --verify upstream/main >/dev/null 2>&1; then
  echo "Upstream Sync Check"
  echo "==================="
  echo "Repository: ${repo_root}"
  echo "Fetch status: ${fetch_status}"
  echo "Fetch note: ${fetch_note}"
  if [[ -n "${fetch_output:-}" ]]; then
    echo "Fetch detail: ${fetch_output}"
  fi
  echo "Upstream ref: upstream/main is unavailable."
  echo
  echo "Open decisions/UPSTREAM_SYNC.md and record why no upstream baseline could be evaluated."
  exit 0
fi

upstream_commits="$(git log main..upstream/main --oneline | wc -l | tr -d ' ')"
divergence="$(git rev-list --left-right --count main...upstream/main | tr '\t' ' ')"
gateway_stat="$(git diff --stat main..upstream/main -- backend/internal/service/gateway_service.go || true)"

echo "Upstream Sync Check"
echo "==================="
echo "Repository: ${repo_root}"
echo "Fetch status: ${fetch_status}"
echo "Fetch note: ${fetch_note}"
if [[ -n "${fetch_output:-}" && "${fetch_status}" != "ok" ]]; then
  echo "Fetch detail: ${fetch_output}"
fi
echo "Upstream new commits vs main: ${upstream_commits}"
echo "Fork divergence (ahead behind): ${divergence}"
echo
echo "gateway_service.go change scope vs upstream/main:"
if [[ -n "${gateway_stat}" ]]; then
  echo "${gateway_stat}"
else
  echo "backend/internal/service/gateway_service.go | no diff"
fi
echo
echo "Next: open decisions/UPSTREAM_SYNC.md and append this practice run."
