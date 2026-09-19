#!/usr/bin/env bash
# Reclaim Docker space left by Harbor trials and task builds. Safe to run while jobs are active:
# only removes stopped containers, dangling/unused volumes, unused task images older than 6h, and trims build cache.
set -u
# never touch images/containers while a Harbor job is active (a prune raced a trial on 2026-09-18 23:58Z)
if pgrep -f "harbor run" >/dev/null 2>&1; then
  docker builder prune -f --keep-storage 4GB >/dev/null 2>&1
  echo "$(date -u +%H:%M) cleanup: harbor active, cache-only; disk_avail=$(df -h / | awk 'NR==2{print $4}')"; exit 0
fi
KEEP_BASE='ladder-base|golang:1.23|harbor-prebuilt'
docker container prune -f >/dev/null 2>&1
docker volume prune -f >/dev/null 2>&1
# task images not used by a running container, older than 6h, not a base image
running=$(docker ps --format '{{.Image}}' | sort -u)
docker images --format '{{.Repository}}:{{.Tag}} {{.ID}} {{.CreatedAt}}' | grep -vE "$KEEP_BASE" | while read -r ref id created; do
  echo "$running" | grep -q "^${ref%%:*}" && continue
  age=$(( $(date +%s) - $(date -d "${created% *}" +%s 2>/dev/null || echo 0) ))
  [ "$age" -gt 86400 ] && docker rmi -f "$id" >/dev/null 2>&1
done
docker image prune -f >/dev/null 2>&1
docker builder prune -f --keep-storage 4GB >/dev/null 2>&1
echo "$(date -u +%H:%M) cleanup: $(docker system df --format '{{.Type}} {{.Size}}' | tr '\n' ';') disk_avail=$(df -h / | awk 'NR==2{print $4}')"
# host-side Go build caches and temp dirs from agent proofs (2026-09-19: 18 GB go-build + 13 GB /tmp in one hour)
export PATH="/usr/local/go/bin:$PATH"
gb=$(du -sm "$HOME/.cache/go-build" 2>/dev/null | cut -f1); [ "${gb:-0}" -gt 8000 ] && go clean -cache >/dev/null 2>&1 && echo "$(date -u +%H:%M) go-build cache cleared (${gb}M)"
find /tmp -maxdepth 1 \( -name 'go-build*' -o -name 'go-link-*' -o -name '*-logs-*' -o -name 'cursor-sdk-bridge-*' \) -mmin +60 -exec rm -rf {} + 2>/dev/null
find /tmp -maxdepth 1 -type f -name '*.log' -mmin +180 -size +50M -delete 2>/dev/null
