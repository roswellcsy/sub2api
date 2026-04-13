#!/bin/bash
# update.sh — one-shot upgrade for zen-codes.com deployment
#
# Usage:
#   bash deploy/prod/update.sh                 # deploy current main
#   bash deploy/prod/update.sh --skip-backup   # skip pre-deploy backup (fast re-deploy)
#
# Preconditions:
#   - ssh alias `racknerd1g` works (or override via SUB2API_REMOTE env)
#   - dev host has pnpm + git + scp + ssh
#   - current main is clean (no uncommitted changes)
#
set -euo pipefail

SUB2API_REMOTE="${SUB2API_REMOTE:-racknerd1g}"
WORKDIR="${WORKDIR:-/opt/api-station}"
DOMAIN="${DOMAIN:-zen-codes.com}"

SKIP_BACKUP=0
for arg in "$@"; do
  case $arg in
    --skip-backup) SKIP_BACKUP=1 ;;
  esac
done

echo "==> Verifying repo state"
cd "$(git rev-parse --show-toplevel)"
git diff --quiet || { echo "ERROR: uncommitted changes. Commit or stash first." >&2; exit 1; }
git fetch origin
BEHIND=$(git rev-list --count HEAD..origin/main)
if [ "$BEHIND" -gt 0 ]; then
  echo "ERROR: local main is behind origin by $BEHIND commits. Pull first." >&2
  exit 1
fi

echo "==> Building frontend dist on dev host"
(cd frontend && pnpm install --frozen-lockfile && pnpm run build)

echo "==> Pushing to origin"
git push origin main

echo "==> Packaging dist"
tar czf /tmp/dist-${DOMAIN}.tar.gz -C backend/internal/web dist

if [ "$SKIP_BACKUP" -eq 0 ]; then
  echo "==> VPS: pre-upgrade backup + rollback tag"
  ssh "$SUB2API_REMOTE" "bash -s" <<'REMOTE'
set -e
# Tag current image for rollback
if docker images weishaw/sub2api:latest --format '{{.Repository}}' | grep -q .; then
  docker tag weishaw/sub2api:latest weishaw/sub2api:rollback-$(date +%Y%m%d-%H%M%S)
  echo "Tagged rollback image"
fi
# Trigger PG backup
if [ -x /etc/cron.daily/pg-backup-zen-codes ]; then
  bash /etc/cron.daily/pg-backup-zen-codes
  ls -lh /backups/ | tail -3
fi
REMOTE
fi

echo "==> VPS: transfer dist + pull + rebuild"
scp /tmp/dist-${DOMAIN}.tar.gz "$SUB2API_REMOTE":/tmp/
ssh "$SUB2API_REMOTE" "bash -s" <<REMOTE
set -e
cd ${WORKDIR}
git pull --ff-only
tar xzf /tmp/dist-${DOMAIN}.tar.gz -C backend/internal/web/
docker build -t weishaw/sub2api:latest -f deploy/prod/Dockerfile.prebuilt . 2>&1 | tail -10
cd deploy
docker compose -f docker-compose.local.yml -f docker-compose.override.yml up -d
REMOTE

echo "==> Waiting for healthy (up to 90s)"
for i in \$(seq 1 18); do
  if curl -fsS --max-time 5 https://${DOMAIN}/health > /dev/null 2>&1; then
    echo "OK @ ${i}"
    curl -sS https://${DOMAIN}/health
    exit 0
  fi
  sleep 5
done
echo "ERROR: health check failed after 90s" >&2
ssh "$SUB2API_REMOTE" "cd ${WORKDIR}/deploy && docker compose -f docker-compose.local.yml -f docker-compose.override.yml logs --tail 40 sub2api"
exit 1
