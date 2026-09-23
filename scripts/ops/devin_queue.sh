#!/usr/bin/env bash
# Queue devin's next two phases, chained, so neither needs babysitting.
#
#   phase A   L2 on the feeder: units devin failed at L0 whose L2 has never run. Measured 41%
#             of these fail L2, and every one that failed and was climbed today certified ABOVE
#             L2 (L3, L4, L5, L5, L6). This is the only pipeline found so far that reliably
#             produces dose-response data rather than another L2 agreement.
#   phase B   the climb: L3-L6 on whatever phase A leaves failing at L2, every cell measured,
#             deep rather than one rung at a time.
#
# Both phases are idempotent. The guard refuses any cell that already has a verdict or is
# already running, so a relaunch picks up only what is left and a crash costs nothing but time.
#
# Usage: devin_queue.sh [wait_for_cohort]
set -u
R=/home/evan/Documents/open_swe_traces_research
WAIT_FOR="${1:-sweep_devin_ladder_all}"
FEEDER=sweep_devin_feeder_L2
cd "$R"
log() { echo "$(date '+%H:%M:%S') $*"; }

runnable() {   # how many units in $1 may devin still run, per the guard, not per the directory
  timeout 400 uv run python - "$1" <<'PY' 2>/dev/null
import sys, os, glob
sys.path.insert(0, "scripts/ops")
import trial_ledger as TL, trial_guard as TG, solver_match as SM
cohort = sys.argv[1]
per = TL.ledger(); n = 0
for d in sorted(glob.glob(f"experiments/dose_response/{cohort}/*/")):
    u = os.path.basename(d.rstrip("/"))
    try:
        if TG.decide(u, per, solver="devin")[0] and SM.decide(u, "devin")[0]:
            n += 1
    except Exception:
        pass
print(n)
PY
}

free_slots() {
  timeout 200 uv run python -c "
import sys; sys.path.insert(0,'scripts/ops'); import slots
o=slots.occupancy('devin'); print(max(0,o['cap']-o['total']))" 2>/dev/null || echo 0
}

# --- wait for the current cohort to drain -------------------------------------------------
log "waiting for $WAIT_FOR to finish"
while :; do
  left=$(runnable "$WAIT_FOR"); left=${left:-0}
  live=$(docker ps --format '{{.Names}}' | grep -c env-main-1 || true)
  [ "$left" -le 0 ] && [ "${live:-0}" -le 0 ] && break
  sleep 300
done
log "$WAIT_FOR drained"
timeout 300 uv run python scripts/ops/reap_orphans.py --apply 2>/dev/null | tail -2

# --- phase A: L2 on the feeder -------------------------------------------------------------
while :; do
  left=$(runnable "$FEEDER"); left=${left:-0}
  [ "$left" -le 0 ] && break
  free=$(free_slots); free=${free:-0}
  if [ "$free" -ge 1 ]; then
    log "phase A: $left unit(s) left in $FEEDER, $free slot(s) free"
    AGENT=devin GUARD_SOLVER=devin bash scripts/ops/sweep_seq.sh "$FEEDER" 1 "$free" \
      >> outputs/supervisor/queue_phaseA.log 2>&1 || true
  else
    sleep 240
  fi
done
log "phase A complete"
timeout 300 uv run python scripts/ops/reap_orphans.py --apply 2>/dev/null | tail -2

# --- phase B: climb whatever now fails L0 and L2 -------------------------------------------
log "phase B: climbing devin's L0+L2 failures to L6"
exec python3 scripts/ops/grok_ladder.py --run --solver devin --select l2fail --max-passes 120
