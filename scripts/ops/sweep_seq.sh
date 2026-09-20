#!/usr/bin/env bash
# Sequential escalation: k=1, then retry only what failed. Same verdict as k=3 "any pass",
# at ~58% of the trials, with ZERO information lost.
#
# Measured on 681 real trials: 30% of them ran AFTER the unit had already passed once.
# A pass is a certificate -- the 2nd and 3rd attempt on a unit that already passed cannot
# change the verdict, and 81% of multi-attempt units were unanimous anyway.
#
# Usage: sweep_seq.sh <sweep dir name> [max-rounds] [concurrency]
set -u
R=/home/evan/Documents/open_swe_traces_research
D="${1:?sweep dir name}"; ROUNDS="${2:-3}"; CONC="${3:-6}"
cd "$R" && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
STAGE="experiments/dose_response/$D"
PEND="$(mktemp -d)/pending"; trap 'rm -rf "$(dirname "$PEND")"' EXIT
cp -a "$STAGE" "$PEND"

# A trial on a unit whose verdict is already on record buys nothing. Measured over
# 1632 trials: 883 of them re-measured a decided unit. The guard is free; a trial is 2.08M tokens.
if [ "${SKIP_GUARD:-0}" != "1" ]; then
  for u in "$PEND"/*/; do
    n=$(basename "$u")
    if ! out=$(uv run python "$R/scripts/ops/trial_guard.py" "$n" 2>/dev/null); then
      echo "   guard: $out"; rm -rf "$u"
    fi
  done
  left=$(find "$PEND" -maxdepth 1 -mindepth 1 -type d | wc -l)
  echo "=== guard kept $left unit(s)"
  [ "$left" -eq 0 ] && { echo "nothing left to trial"; exit 0; }
fi

# GATE: an L2 unit may not be trialled without an audit row. The audit costs ~6k tokens and
# names the missing commitments; the trial costs 2.08M and returns one bit. On 2026-09-20 ten
# units went to Composer ungated and the audit then found client-go-replicaselector-L2 had 12
# missing commitments and 2 conflicts -- a guaranteed failure, pulled mid-flight.
GAPS="$R/experiments/dose_response/audit/gap_read.jsonl"
ungated=""
for u in "$PEND"/*/; do
  [ -d "$u" ] || continue
  n=$(basename "$u")
  case "$n" in *-L0|*-L1) continue;; esac          # no contract at L0/L1; TOO-EASY is the gate there
  grep -q "\"unit\": \"$n\"" "$GAPS" 2>/dev/null || ungated="$ungated $n"
done
if [ -n "$ungated" ]; then
  echo "!! these L2 units have no audit row -- auditing before spending a trial:$ungated"
  for n in $ungated; do
    uv run python "$R/scripts/ops/contract_gap_read.py" "$GAPS" "$PEND/$n" || true
  done
  # drop anything the audit says cannot flip as written
  for n in $ungated; do
    v=$(python3 -c "
import json,sys
for l in open('$GAPS'):
    d=json.loads(l)
    if d['unit']=='$n':
        print('DEFECT' if (d['missing'] or d['conflicts']) else 'OK')
        break
" 2>/dev/null)
    if [ "$v" = "DEFECT" ]; then
      # WARN by default. A REPAIRED unit still shows residual gaps (the loop converges on the
      # shadow passing, not on zero gaps), so dropping on "any gap" vetoes exactly the units we
      # staged to test. Set GATE_DROP=1 to enforce.
      if [ "${GATE_DROP:-0}" = "1" ]; then
        echo "   dropping $n (GATE_DROP=1)"; rm -rf "$PEND/$n"
      else
        echo "   note: $n has audit gaps — trialling anyway"
      fi
    fi
  done
fi

for round in $(seq 1 "$ROUNDS"); do
  n=$(find "$PEND" -maxdepth 1 -mindepth 1 -type d | wc -l)
  [ "$n" -eq 0 ] && { echo "round $round: nothing left"; break; }
  echo "=== round $round: $n units still unflipped"
  docker network prune -f >/dev/null 2>&1
  harbor run --path "$PEND" --agent cursor-cli --model cursor/composer-2.5 \
    --n-concurrent "$CONC" --n-attempts 1 --max-retries 1 \
    --jobs-dir experiments/dose_response/jobs --job-name "${D}_r${round}" --yes
  "$R/scripts/ops/post_sweep.sh" "${D}_r${round}"
  # drop every unit that passed this round
  for t in "experiments/dose_response/jobs/${D}_r${round}"/*/; do
    [ -f "$t/result.json" ] || continue
    rew=$(python3 -c "
import json,sys
d=json.load(open('$t/result.json'))
print((d.get('verifier_result') or {}).get('rewards',{}).get('reward'))" 2>/dev/null)
    [ "$rew" = "1.0" ] || continue
    u=$(basename "$t" | sed 's/__.*//')
    rm -rf "$PEND/$u"
  done
done
echo "=== unflipped after $ROUNDS rounds:"; ls "$PEND" 2>/dev/null
