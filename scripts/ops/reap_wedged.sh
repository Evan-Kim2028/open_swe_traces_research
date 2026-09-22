#!/usr/bin/env bash
# Kill only containers that are provably NOT working. Age alone is not evidence.
#
# Three trials were killed today on age alone while still making progress, and harbor then
# RETRIED each one, extending the job instead of ending it.
#
# "0% CPU and no log output" was the replacement rule, and it was still wrong -- it killed
# nine live trials, four of them confirmed L2s (rootval, httpresp, defval, methodval), each
# costing 20-50 minutes of a Devin slot capped at four. Devin is a CLOUD agent: the model
# runs on Devin's servers, so a container that is thinking shows 0% local CPU and writes
# nothing to rollout.log for minutes at a time. It is not idle, it is waiting on a socket.
#
# Network I/O is the signal that separates the two. A container talking to api.devin.ai
# moves bytes the whole time it is alive; a wedged one moves none. All three must be flat
# -- CPU, log bytes, and network -- before anything is killed.
#
# Usage: reap_wedged.sh [min_age_min] [sample_sec] [--dry-run]   (flags in any position)
#
# --dry-run USED TO BE POSITIONAL, `DRY="${3:-}"`, and that made the safe invocation the
# dangerous one. `reap_wedged.sh --dry-run` -- the form in the monitoring runbook, run every
# thirty minutes -- set MINAGE="--dry-run" and left DRY empty, so it was a LIVE KILL RUN.
# Worse, `[ "$age" -lt "--dry-run" ]` errors instead of returning true, so `&& continue`
# never fired and the minimum-age floor was disabled too: every running container became a
# candidate, Devin trials included, which are the one thing that must never be killed. It
# only ever printed "keeping" because the containers it sampled were genuinely busy. The
# integer-expression errors on stderr were the sole evidence, and they look like noise.
#
# Parse flags by name, then REFUSE anything unrecognised. A monitor that silently reinterprets
# its own safety flag as a threshold is worse than one that crashes.
set -u
declare -A c0 l0 cpu n0
DRY=""; POS=()
for a in "$@"; do
  case "$a" in
    --dry-run|-n) DRY="--dry-run" ;;
    -*) echo "reap_wedged.sh: unknown flag '$a'; usage: reap_wedged.sh [min_age_min] [sample_sec] [--dry-run]" >&2; exit 2 ;;
    *)  POS+=("$a") ;;
  esac
done
MINAGE="${POS[0]:-45}"; SAMPLE="${POS[1]:-120}"
for v in "$MINAGE" "$SAMPLE"; do
  case "$v" in
    ''|*[!0-9]*) echo "reap_wedged.sh: min_age_min and sample_sec must be integers, got '$v'" >&2; exit 2 ;;
  esac
done
CPU_FLOOR=1.0     # percent; a thinking agent still moves more than this
now=$(date +%s)

# Orphaned sidecars first: killing a harbor run (or a crash) leaves the per-trial egress
# sidecar behind with no env-main to serve. Five accumulated from one SIGKILL at 06:16,
# each holding a container slot and confusing every count that greps docker ps. A sidecar
# whose main container is gone is unambiguously dead weight - no sampling needed.
mains=$(docker ps --format '{{.Names}}' | grep 'env-main-1' | sed 's/__env-main-1//')
for c in $(docker ps --format '{{.Names}}' | grep 'egress-control-sidecar'); do
  stem=${c%%__env-harbor-docker-egress-control-sidecar-1}
  if ! echo "$mains" | grep -qx "$stem"; then
    if [ "$DRY" = "--dry-run" ]; then
      echo "  WOULD KILL orphan sidecar $c (no env-main)"
    else
      echo "  killing orphan sidecar $c (no env-main)"
      docker kill "$c" >/dev/null 2>&1
    fi
  fi
done

# Bytes in+out, as a single integer. docker stats prints "1.2MB / 345kB"; the units are
# what matter far less than whether the number moved at all, so normalise to bytes.
netio() {
  docker stats --no-stream --format '{{.NetIO}}' "$1" 2>/dev/null | awk '
    function b(v,  n,u){ n=v+0; u=v; gsub(/[0-9.]/,"",u)
      if(u ~ /^kB/) return n*1000; if(u ~ /^MB/) return n*1000000
      if(u ~ /^GB/) return n*1000000000; return n }
    { split($0, a, " / "); printf "%.0f", b(a[1]) + b(a[2]) }'
}

for c in $(docker ps --format '{{.Names}}' | grep 'env-main'); do
  started=$(docker inspect -f '{{.State.StartedAt}}' "$c" 2>/dev/null) || continue
  age=$(( (now - $(date -d "$started" +%s 2>/dev/null || echo "$now")) / 60 ))
  [ "$age" -lt "$MINAGE" ] && continue
  c0[$c]=$(docker inspect -f '{{.State.Pid}}' "$c" 2>/dev/null)
  l0[$c]=$(docker logs "$c" 2>&1 | wc -c)
  n0[$c]=$(netio "$c")
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
  net=$(( $(netio "$c") - ${n0[$c]:-0} ))
  # A NEGATIVE delta means the counter moved in a way we cannot read — a reset, or an
  # overflow in the sampler (docker stats NetIO past 2GB wrapped awk's %d to negative,
  # and a wrapped sample reads exactly like an idle one). Only an exactly flat counter
  # is evidence of silence.
  if [ "$used" -eq 0 ] && [ "$grew" -eq 0 ] && [ "$net" -eq 0 ]; then
    if [ "$DRY" = "--dry-run" ]; then
      echo "  WOULD KILL $c — 0% cpu, 0 log bytes and 0 net bytes over ${SAMPLE}s"
    else
      echo "  killing $c — 0% cpu, 0 log bytes and 0 net bytes over ${SAMPLE}s (wedged)"
      docker kill "$c" >/dev/null 2>&1
    fi
  else
    echo "  keeping $c — cpu=${used} log +${grew}B net +${net}B (still working)"
  fi
done
