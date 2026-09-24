#!/usr/bin/env bash
# Keep a cohort's solver at its cap, when the cohort was launched narrower than the cap.
#
# sweep_seq fixes --n-concurrent at launch. A cohort started while only two slots were free
# keeps using two forever, so as older trials drain the freed slots go idle -- which is
# exactly what pausing the screening queue was supposed to avoid. Relaunching the same
# cohort is safe and idempotent: trial_guard refuses every unit that already has a verdict,
# so a top-up sweep picks up only what is left.
#
# Usage: cohort_topup.sh <cohort> <agent> <solver> [min_free]
set -u
R=/home/evan/Documents/open_swe_traces_research
C="${1:?cohort}"; AG="${2:?agent}"; SV="${3:?solver}"; MINFREE="${4:-2}"
cd "$R"
while true; do
  # Any unit left that this solver may still run? Ask the guard, not the directory listing.
  left=$(timeout 300 uv run python - "$C" "$SV" <<'PY' 2>/dev/null
import sys, os, glob
sys.path.insert(0, "scripts/ops")
import trial_ledger as TL, trial_guard as TG, solver_match as SM
cohort, solver = sys.argv[1], sys.argv[2]
agent = {"devin": "devin", "composer": "cursor-cli", "grok": "grok-build"}[solver]
per = TL.ledger(); n = 0
for d in sorted(glob.glob(f"experiments/dose_response/{cohort}/*/")):
    u = os.path.basename(d.rstrip("/"))
    try:
        if TG.decide(u, per, solver=solver)[0] and SM.decide(u, agent)[0]:
            n += 1
    except Exception:
        pass
print(n)
PY
)
  left=${left:-0}
  [ "$left" -le 0 ] && { echo "$(date '+%H:%M:%S') $C: nothing left for $SV"; exit 0; }
  free=$(timeout 200 uv run python -c "
import sys; sys.path.insert(0,'scripts/ops'); import slots
o=slots.occupancy('$SV'); print(max(0,o['cap']-o['total']))" 2>/dev/null || echo 0)
  free=${free:-0}
  if [ "$free" -ge "$MINFREE" ]; then
    echo "$(date '+%H:%M:%S') $C: $left unit(s) left, $free slot(s) free — topping up"
    AGENT="$AG" GUARD_SOLVER="$SV" bash scripts/ops/sweep_seq.sh "$C" 1 "$free" \
      >> "outputs/supervisor/topup_${C}.log" 2>&1 || true
  fi
  sleep 300
done
