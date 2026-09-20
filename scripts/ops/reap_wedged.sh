#!/usr/bin/env bash
# Kill only containers that are provably NOT working. Age alone is not evidence.
#
# Three trials were killed today on age alone while still making progress, and harbor then
# RETRIED each one, extending the job instead of ending it. A wedged container is one that,
# over a real sampling window, does no CPU work AND writes no log output. Both must hold.
#
# Usage: reap_wedged.sh [min_age_min] [sample_sec] [--dry-run]
set -u
declare -A c0 l0 cpu
MINAGE="${1:-45}"; SAMPLE="${2:-45}"; DRY="${3:-}"
CPU_FLOOR=1.0     # percent; a thinking agent still moves more than this
now=$(date +%s)

for c in $(docker ps --format '{{.Names}}' | grep 'env-main'); do
  started=$(docker inspect -f '{{.State.StartedAt}}' "$c" 2>/dev/null) || continue
  age=$(( (now - $(date -d "$started" +%s 2>/dev/null || echo "$now")) / 60 ))
  [ "$age" -lt "$MINAGE" ] && continue
  c0[$c]=$(docker inspect -f '{{.State.Pid}}' "$c" 2>/dev/null)
  l0[$c]=$(docker logs "$c" 2>&1 | wc -c)
done
# an empty associative array trips `set -u` on ${#a[@]} in bash < 5.1; test the expansion
if [ -z "${c0[*]:-}" ]; then echo "reap: no container older than ${MINAGE}m"; exit 0; fi

echo "reap: sampling ${#c0[@]} container(s) for ${SAMPLE}s"

for c in "${!c0[@]}"; do cpu[$c]=0; done
end=$(( $(date +%s) + SAMPLE ))
while [ "$(date +%s)" -lt "$end" ]; do
  while IFS=$'\t' read -r name pct; do
    [ -n "${c0[$name]+x}" ] || continue
    v=${pct%\%}; v=${v%.*}
    [ "${v:-0}" -gt 0 ] && cpu[$name]=$(( ${cpu[$name]} + v ))
  done < <(docker stats --no-stream --format '{{.Name}}\t{{.CPUPerc}}' 2>/dev/null)
  sleep 10
done

for c in "${!c0[@]}"; do
  l1=$(docker logs "$c" 2>&1 | wc -c)
  grew=$(( l1 - ${l0[$c]} ))
  used=${cpu[$c]}
  if [ "$used" -eq 0 ] && [ "$grew" -eq 0 ]; then
    if [ "$DRY" = "--dry-run" ]; then
      echo "  WOULD KILL $c — 0% cpu and 0 log bytes over ${SAMPLE}s"
    else
      echo "  killing $c — 0% cpu and 0 log bytes over ${SAMPLE}s (wedged)"
      docker kill "$c" >/dev/null 2>&1
    fi
  else
    echo "  keeping $c — cpu samples=${used} log grew ${grew}B (still working)"
  fi
done
