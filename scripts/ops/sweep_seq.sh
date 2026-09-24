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

# Record pre-trial features while the staged copy still exists. 74 units are known
# too-easy but only 27 still had their tests on disk when we went to calibrate.
uv run python "$R/scripts/ops/unit_features.py" "$PEND" >/dev/null 2>&1 || true

# A trial on a unit whose verdict is already on record buys nothing. Measured over
# 1632 trials: 883 of them re-measured a decided unit. The guard is free; a trial is 2.08M tokens.
# The ladder is per solver now, so the guard has to be told who is asking: composer's
# rungs and devin's rungs are two different curves and a rung answered by one is still
# open for the other. solver_match.py owns the agent->solver mapping; ask it rather than
# keeping a second copy of the table here.
GUARD_SOLVER=$(uv run python -c "import sys;sys.path.insert(0,'$R/scripts/ops');import solver_match;print(solver_match.solver_for('${AGENT:-cursor}'))" 2>/dev/null || echo composer)

if [ "${SKIP_GUARD:-0}" != "1" ]; then
  for u in "$PEND"/*/; do
    n=$(basename "$u")
    if ! out=$(uv run python "$R/scripts/ops/trial_guard.py" "$n" --solver "$GUARD_SOLVER" 2>/dev/null); then
      echo "   guard: $out"; rm -rf "$u"
    fi
  done
  # Free deterministic gate, second. Controlled measurement 2026-09-20 over 229 units:
  #   parallel k=3, ungated     9.0 trials/certified
  #   sequential k=1, ungated   7.7   (sequencing alone buys 14%)
  #   sequential k=1, GATED     3.2   (the gate buys a further 58%)
  # A trial is ~1.2M tokens and a CPU-bound container slot; the linter is 0 tokens.
  # BLOCK findings only -- WARN and INFO never drop a unit, because a repaired unit
  # legitimately still carries residual gaps.
  for u in "$PEND"/*/; do
    n=$(basename "$u")
    blocks=$(uv run python "$R/scripts/ops/task_lint.py" "$u" 2>/dev/null | grep -c '\[BLOCK\]' || true)
    if [ "${blocks:-0}" -gt 0 ]; then
      echo "   lint: dropping $n ($blocks BLOCK finding(s))"
      uv run python "$R/scripts/ops/task_lint.py" "$u" 2>/dev/null | grep '\[BLOCK\]' | sed 's/^/        /'
      rm -rf "$u"
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
# GATE: a unit at L2 or above may only be trialled by a solver that FAILED it lower
# down. Routing by free slot instead produced 25 cross-solver certificates of 193 —
# attachsvc (composer failed L0, devin passed L2), channelver (the reverse) — where the
# flip may record only that the second model is stronger, not that the contract carries
# the affordance. Same rule escalate.py applies to L3-L6, at the rung where most
# certificates are earned. Costs the L2 overflow valve; a certificate that does not
# isolate the affordance is not worth the slot it saves.
#
# ONE invocation for the whole cohort. Per-unit `uv run python` spends more time starting
# interpreters than deciding, and the gate is meant to be the cheap step. Dropping here
# touches only $PEND, a mktemp copy — the staged unit itself is never removed.
xs=""
for u in "$PEND"/*/; do
  [ -d "$u" ] || continue
  case "$(basename "$u")" in *-L0|*-L1) continue;; esac
  xs="$xs $u"
done
if [ -n "$xs" ]; then
  # shellcheck disable=SC2086
  ( cd "$R" && uv run python scripts/ops/solver_match.py "${AGENT:-cursor}" $xs 2>/dev/null ) \
    | while read -r verdict name rest; do
        [ "$verdict" = "DROP" ] || continue
        echo "   cross-solver: dropping $name ($rest)"
        rm -rf "$PEND/$name"
      done
fi

GAPS="$R/experiments/dose_response/audit/gap_read.jsonl"
ungated=""
for u in "$PEND"/*/; do
  [ -d "$u" ] || continue
  n=$(basename "$u")
  case "$n" in *-L0|*-L1) continue;; esac          # no contract at L0/L1; TOO-EASY is the gate there
  # A row whose answer is __ERROR__ is a FAILED audit, not a passed one. It carries
  # missing=0 conflicts=0, so the DEFECT check below reads it as a perfect contract.
  # Treat it as absent so the unit is audited for real.
  if grep "\"unit\": \"$n\"" "$GAPS" 2>/dev/null | grep -qv '"answer": "__ERROR__'; then
    continue
  fi
  # An escalation rung carries the SAME contract prose as its L2 twin; L3+ only appends
  # hidden test names. Re-auditing it asks the same question again at ~90s per unit, and
  # a 38-unit escalation cohort therefore held eight reserved Composer slots idle for the
  # best part of an hour before its first trial. Inherit the L2 row instead.
  case "$n" in
    *-L[3-6])
      twin="${n%-L[3-6]}-L2"
      row=$(grep "\"unit\": \"$twin\"" "$GAPS" 2>/dev/null | tail -1)
      if [ -n "$row" ]; then
        echo "$row" | sed "s/\"unit\": \"$twin\"/\"unit\": \"$n\"/" >> "$GAPS"
        echo "   $n inherits the audit row of $twin"
        continue
      fi
      ;;
  esac
  ungated="$ungated $n"
done
if [ -n "$ungated" ]; then
  echo "!! these L2 units have no audit row -- auditing before spending a trial:$ungated"
  # One invocation for the whole cohort: it parallelises internally. Previously this
  # spawned a process per unit, serially, so a 12-unit cohort spent ~8 minutes gating
  # while every container slot sat idle.
  # shellcheck disable=SC2086
  ( cd "$R" && uv run python "$R/scripts/ops/contract_gap_read.py" "$GAPS" \
      $(for n in $ungated; do echo "$PEND/$n"; done) ) || true
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
  # Re-gate between rounds. The guard runs at staging, but rounds 2 and 3 bypassed it:
  # sweep_seq drops passers and retries failures, which is right at L2 (retry until it
  # flips) and wrong at L0, where a FAILURE is the verdict we want. sweep_unknown spent
  # 9 of 12 trials in one window re-running units already known hard at L0.
  if [ "$round" -gt 1 ]; then
    for u in "$PEND"/*/; do
      [ -d "$u" ] || continue
      un=$(basename "$u")
      if ! g=$(uv run python "$R/scripts/ops/trial_guard.py" "$un" --solver "$GUARD_SOLVER" 2>/dev/null); then
        echo "   re-gate: $g"; rm -rf "$u"
      fi
    done
    n=$(find "$PEND" -maxdepth 1 -mindepth 1 -type d | wc -l)
    [ "$n" -eq 0 ] && { echo "round $round: all remaining units already decided"; break; }
  fi
  echo "=== round $round: $n units still unflipped"
  docker network prune -f >/dev/null 2>&1
  # Solver is swappable so a second sweep can run on a different token quota in parallel.
  # AGENT=grok-build MODEL=grok-4.6 GROK_EFFORT=high uses Grok instead of Composer.
  if [ "${AGENT:-cursor-cli}" = "devin" ]; then
    # Devin is markedly slower per trial than Composer, so give it real headroom: a first
    # run was cut off at 15 minutes mid-solve and the resulting CancelledError looked like
    # a failure. Two settings are load-bearing and neither is obvious:
    #   --model devin/swe-2-max   without the devin/ prefix harbor's slug split yields a
    #                             name the CLI rejects as "Unknown model"
    #   DEVIN_API_SERVER_URL      harbor writes only windsurf_api_key into the container's
    #                             credentials.toml; without the server URL the CLI's model
    #                             list comes back EMPTY, which also presents as a bad slug
    DKEY=$(grep windsurf_api_key /home/evan/.local/share/devin/credentials.toml 2>/dev/null | cut -d'"' -f2)
    [ -z "$DKEY" ] && { echo "no devin credentials"; exit 1; }
    AGENT_KWARGS=(--agent devin --model "${MODEL:-devin/swe-2-max}"
                  --ae DEVIN_API_KEY="$DKEY"
                  --ae DEVIN_API_SERVER_URL=https://server.codeium.com
                  --agent-timeout-multiplier "${DEVIN_TIME_MULT:-4.0}")
  elif [ "${AGENT:-cursor-cli}" = "grok-build" ]; then
    GKEY="$(bash "$R/scripts/ops/grok_key.sh")" || { echo "grok token unavailable"; exit 1; }
    # The CLI authenticates from ~/.grok/auth.json, NOT from XAI_API_KEY -- it rejects a
    # grok.com session token handed to it as that variable ("Not signed in"), which is why
    # 21 of the first 26 grok trials errored. Verified: isolated HOME + XAI_API_KEY fails,
    # isolated HOME + auth.json runs grok-4.7 fine. So send the file's contents and let
    # the patched agent write it inside the container (scripts/ops/grok_build_patch.py --
    # re-run after any `uv tool upgrade harbor`, which silently reverts it).
    GAUTH="$(cat /home/evan/.grok/auth.json 2>/dev/null)" || { echo "grok auth.json unreadable"; exit 1; }
    if ! uv run python "$R/scripts/ops/grok_build_patch.py" --check >/dev/null 2>&1; then
      echo "grok-build agent is NOT patched for host auth — run scripts/ops/grok_build_patch.py"; exit 1
    fi
    AGENT_KWARGS=(--agent grok-build --model "${MODEL:-grok-4.6}"
                  --ak reasoning_effort="${GROK_EFFORT:-high}"
                  --ae XAI_API_KEY="$GKEY"
                  --ae GROK_AUTH_JSON="$GAUTH")
  else
    AGENT_KWARGS=(--agent "${AGENT:-cursor-cli}" --model "${MODEL:-cursor/composer-2.5}")
  fi
  # Job names restarted at _r1 on every relaunch, so re-running a cohort that had already
  # completed died instantly with "already exists and cannot be resumed with a different
  # config" - and now that sweep locks release as soon as their owner exits, relaunching a
  # finished cohort is the normal case, not an edge case. Suffix on collision only, so the
  # usual first run keeps its familiar name. trial_guard's rsplit("_r", 1) stem still
  # resolves to the cohort, so re-trial protection is unaffected.
  JOB="${D}_r${round}"
  if [ -d "experiments/dose_response/jobs/$JOB" ]; then
    JOB="${D}_r${round}_$(date +%H%M%S)"
  fi
  # ADMISSION GATE. The cap is enforced here, at the one place that actually launches trials,
  # because enforcing it in the callers did not work: it broke three times in one evening, each
  # time down a different path. grok_ladder never asked (it was written for grok, which has no
  # cap) and launched seven at once against four. cohort_topup and a sweep asked at the same
  # moment, both were told "four free", and both launched -- the guard's cap counts VERDICTS and
  # an in-flight trial has none. And killing a harbor run left its containers running, spending
  # quota while invisible to an occupancy that summed declared concurrency, so the count said
  # 4/4 with nine up. 15 trials errored in the resulting throttle.
  #
  # Five things launch trials and every one of them decided its own concurrency by asking an
  # advisory helper. So the limit moves to the choke point, under a lock so two launchers cannot
  # both read the same free count, and clamps to what is MEASURED free rather than what the
  # caller asked for.
  # One lock per solver: the caps are per solver, so a Devin launch must never make a
  # Composer launch wait, or the reverse.
  SOLVER_FOR_CAP="${GUARD_SOLVER:-composer}"
  GATE="outputs/supervisor/launch.$SOLVER_FOR_CAP.lock"
  mkdir -p "$(dirname "$GATE")"
  exec 9>"$GATE"
  flock 9 || true                       # serialise the decision, not the sweep
  if [ "$SOLVER_FOR_CAP" = "devin" ]; then
    FREE=$(timeout 120 uv run python -c "
import sys; sys.path.insert(0,'scripts/ops'); import slots
o=slots.occupancy('devin'); print(max(0, o['cap']-o['total']))" 2>/dev/null || echo 1)
    FREE=${FREE:-1}
    if [ "$FREE" -le 0 ]; then
      echo "   admission: devin at cap, not launching this round"
      exec 9>&-
      sleep 120
      continue
    fi
    if [ "$CONC" -gt "$FREE" ]; then
      echo "   admission: clamping concurrency $CONC -> $FREE (measured free devin slots)"
      CONC="$FREE"
    fi
  fi
  harbor run --path "$PEND" "${AGENT_KWARGS[@]}" \
    --n-concurrent "$CONC" --n-attempts 1 --max-retries 1 \
    --jobs-dir experiments/dose_response/jobs --job-name "$JOB" --yes 9>&- &
  # 9>&-: harbor must not inherit the lock fd. It did, so the lock stayed held for the
  # whole RUN, not the admission decision, and every other launch queued behind it.
  HPID=$!
  sleep 20                              # let harbor claim its slots before releasing the lock
  exec 9>&-
  wait "$HPID"
  "$R/scripts/ops/post_sweep.sh" "$JOB"
  # drop every unit that passed this round
  for t in "experiments/dose_response/jobs/$JOB"/*/; do
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
# dirs only: `ls "$PEND"` listed a stray README.md as an unflipped unit. Cosmetic -
# every real enumeration above uses "$PEND"/*/ - but it made the report lie.
echo "=== unflipped after $ROUNDS rounds:"; for d in "$PEND"/*/; do [ -d "$d" ] && basename "$d"; done 2>/dev/null
