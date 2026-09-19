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
