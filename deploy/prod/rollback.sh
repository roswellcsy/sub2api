#!/bin/bash
# rollback.sh — emergency rollback to previous image
#
# Usage:
#   bash deploy/prod/rollback.sh                  # list rollback tags, pick newest
#   bash deploy/prod/rollback.sh <tag>            # specify exact rollback tag
#   bash deploy/prod/rollback.sh --with-db <tag>  # also restore latest DB backup
#
set -euo pipefail

SUB2API_REMOTE="${SUB2API_REMOTE:-racknerd1g}"
WORKDIR="${WORKDIR:-/opt/api-station}"

WITH_DB=0
TAG=""
for arg in "$@"; do
  case $arg in
    --with-db) WITH_DB=1 ;;
    --help) echo "Usage: $0 [--with-db] [<tag>]"; exit 0 ;;
    *) TAG=$arg ;;
  esac
done

if [ -z "$TAG" ]; then
  echo "Available rollback tags on VPS:"
  ssh "$SUB2API_REMOTE" "docker images weishaw/sub2api --format '{{.Tag}} {{.CreatedAt}}' | grep rollback | head -5"
  read -p "Enter tag to rollback to: " TAG
fi

[ -n "$TAG" ] || { echo "no tag given" >&2; exit 1; }

echo "==> Rolling back to $TAG (with-db=$WITH_DB)"
ssh "$SUB2API_REMOTE" "bash -s" <<REMOTE
set -e
cd ${WORKDIR}/deploy

docker tag weishaw/sub2api:${TAG} weishaw/sub2api:latest
echo "image retagged"

if [ ${WITH_DB} -eq 1 ]; then
  LATEST_BACKUP=\$(ls -t /backups/sub2api-*.sql.gz | head -1)
  echo "Stopping sub2api for DB restore..."
  docker compose -f docker-compose.local.yml -f docker-compose.override.yml stop sub2api
  echo "Restoring from \$LATEST_BACKUP"
  gunzip -c "\$LATEST_BACKUP" | docker exec -i sub2api-postgres psql -U sub2api sub2api
fi

docker compose -f docker-compose.local.yml -f docker-compose.override.yml up -d --force-recreate sub2api
REMOTE

sleep 15
if curl -fsS --max-time 5 https://zen-codes.com/health > /dev/null 2>&1; then
  echo "Rollback OK"
  curl -sS https://zen-codes.com/health
else
  echo "WARN: health check failed, investigate immediately"
  exit 1
fi
