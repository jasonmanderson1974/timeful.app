#!/usr/bin/env bash
#
# One-command deploy for VM-public. Run from the repo root ON THE VM:
#
#     ./deploy.sh
#
# It pulls the latest main, rebuilds only the service(s) whose code actually
# changed, recreates the server when the frontend changed (the Go server
# registers the frontend's static files at startup, so it must restart to serve
# the new hashed filenames), health-checks, and prunes the Docker build cache
# (the 30G disk fills quickly otherwise — see the memory notes).
#
# Docs/config-only changes (e.g. TODO.md) skip the rebuild entirely.
set -euo pipefail

cd "$(dirname "$0")"

echo "==> Pulling latest main..."
BEFORE=$(git rev-parse HEAD)
git pull --ff-only
AFTER=$(git rev-parse HEAD)

if [ "$BEFORE" = "$AFTER" ]; then
  echo "Already up to date at ${AFTER:0:7} — nothing to deploy."
  exit 0
fi
echo "==> ${BEFORE:0:7} -> ${AFTER:0:7}"

CHANGED=$(git diff --name-only "$BEFORE" "$AFTER")
SERVER_CHANGED=false
FRONTEND_CHANGED=false
if echo "$CHANGED" | grep -q '^server/';   then SERVER_CHANGED=true;   fi
if echo "$CHANGED" | grep -q '^frontend/'; then FRONTEND_CHANGED=true; fi

if ! $SERVER_CHANGED && ! $FRONTEND_CHANGED; then
  echo "==> No server/ or frontend/ changes (docs/config only) — no rebuild needed."
else
  TARGETS=()
  if $SERVER_CHANGED;   then TARGETS+=("server");   fi
  if $FRONTEND_CHANGED; then TARGETS+=("frontend"); fi
  echo "==> Rebuilding: ${TARGETS[*]}"
  docker compose up -d --build "${TARGETS[@]}"

  # Frontend rebuilt -> new hashed asset filenames. The server registers static
  # routes at startup, so recreate it to serve the new dist.
  if $FRONTEND_CHANGED; then
    echo "==> Frontend changed — recreating server to serve the new dist..."
    docker compose up -d --force-recreate server
  fi
fi

echo "==> Waiting for health..."
sleep 4
CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3002/api/health || echo "000")
if [ "$CODE" != "200" ]; then
  echo "!! Health check returned ${CODE} — investigate (build may have failed; old container may still be up)."
  docker compose ps
  exit 1
fi
echo "==> Health OK (200)."

echo "==> Pruning Docker build cache..."
docker builder prune -af >/dev/null
df -h / | tail -1

echo "==> Deployed to ${AFTER:0:7}."
