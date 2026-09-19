#!/usr/bin/env bash
# dq.sh -- devin queue runner with liveness, outcome tracking and a circuit breaker.
set -u
D=/home/evan/devin-tasks; Q=$D/queue; R=$D/running; DONE=$D/done; FAILED=$D/failed
HB=$D/dq.heartbeat; RES=$D/dq.results.tsv; HALT=$D/DQ_HALTED
# MAXN 3 -> 4: 2026-09-18 capacity trial. Watch VPS load and lor-api p95;
# agents have starved that API before. Revert to 3 if p95 degrades.
MAXN="${MAXN:-4}"; POLL="${POLL:-60}"; BREAK_AFTER="${BREAK_AFTER:-3}"; MIN_OK_SECS="${MIN_OK_SECS:-120}"
# Rate-limit backoff for the MAXN=4 trial (2026-09-18).
# A RATE_LIMIT verdict means 4 concurrent devin sessions is too many. Pause
# launches for COOLDOWN_S, then run at 3 permanently — do not creep back to 4,
# because the next trial should be a deliberate decision, not a side effect of
# a restart. State lives in a file so it survives a runner restart; the
# supervisor restarting dq.sh must not silently undo the back-off.
SAFE_MAXN="${SAFE_MAXN:-3}"
COOLDOWN_S="${DQ_RATE_COOLDOWN_S:-1800}"   # 30m
RL_STATE="$D/.dq_rate_limited"
# Only count rate limits that happen DURING this trial. A stale RATE_LIMIT from
# an earlier run at MAXN=3 tripped this instantly on first start and would have
# pinned us to the safe cap without ever testing 4.
TRIAL_START="$D/.dq_trial_start"
[ -f "$TRIAL_START" ] || date -Is > "$TRIAL_START"
mkdir -p "$Q" "$R" "$DONE" "$FAILED"; [ -f "$RES" ] || printf 'ts\tname\tmodel\tdur_s\texit\tverdict\tdetail\n' > "$RES"

live() {
  ps -eo args 2>/dev/null | grep -oE '\-\-prompt-file [^ ]+' | awk '{print $2}' | sort -u \
    | while read -r f; do [ -f "$f" ] && echo "$f"; done | wc -l
}

# Classify a finished job from its output file.
classify() { # <out> <dur> <exit>
  local out="$1" dur="$2" rc="$3"
  if grep -qiE 'rate.?limit|too many requests|429|quota exceeded|overloaded' "$out" 2>/dev/null; then echo "RATE_LIMIT"; return; fi
  if grep -qiE 'unauthorized|authentication failed|not logged in|invalid api key|401' "$out" 2>/dev/null; then echo "AUTH"; return; fi
  if grep -qE 'https://github\.com/[^ ]+/pull/[0-9]+|NO PR NEEDED|MERGED [0-9a-f]{7}' "$out" 2>/dev/null; then echo "OK"; return; fi
  if [ "$dur" -lt "$MIN_OK_SECS" ]; then echo "FAST_FAIL"; return; fi
  if [ "$rc" -ne 0 ]; then echo "ERROR"; return; fi
  echo "NO_PR"
}

launch() {
  local brief="$1" model="$2" name; name=$(basename "$brief" .md)
  mv "$brief" "$R/$name.md" 2>/dev/null || return 1
  setsid nohup bash -c "
    s=\$(date +%s)
    cd /home/evan/Documents
    timeout 7200 devin --model $model --permission-mode dangerous -p --prompt-file $R/$name.md > $D/$name.devin.out 2>&1
    rc=\$?; e=\$(date +%s); dur=\$((e-s))
    echo \"exit \$rc\" >> $D/$name.devin.out
    v=\$($D/dq-classify.sh $D/$name.devin.out \$dur \$rc)
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \"\$(date -Is)\" '$name' '$model' \"\$dur\" \"\$rc\" \"\$v\" \"\$(grep -oE 'https://github.com/[^ ]+/pull/[0-9]+' $D/$name.devin.out | tail -1)\" >> $RES
    if [ \"\$v\" = OK ]; then mv $R/$name.md $DONE/$name.md 2>/dev/null; else mv $R/$name.md $FAILED/$name.md 2>/dev/null; fi
  " >/dev/null 2>&1 </dev/null &
  echo "[dq] launched $name ($model)"
}

consecutive_bad() { tail -n 6 "$RES" | awk -F'\t' 'NR>0{print $6}' | tac | awk '/^OK$/{exit} /^(RATE_LIMIT|AUTH|FAST_FAIL|ERROR|NO_PR|ASKED)$/{c++} END{print c+0}'; }


review_exists() {
  local n="$1" p
  for p in "$D/r-$n.md" "$Q/90-review-$n.md" "$DONE/90-review-$n.md" "$R/90-review-$n.md" "$FAILED/90-review-$n.md"; do
    [ -f "$p" ] && return 0
  done
  grep -ls "#$n\b" "$Q"/89-review-batch-*.md "$R"/89-review-batch-*.md "$DONE"/89-review-batch-*.md "$FAILED"/89-review-batch-*.md 2>/dev/null | grep -q . && return 0
  return 1
}

scan_for_reviews() {
  # Collect every uncovered PR first, then decide single-vs-batch.
  # Batching is not slot economy: a per-PR reviewer cannot see two
  # independently-correct changes combining badly (contradictory settings in a
  # shared config, duplicate guards under different names, one PR invalidating
  # another's thresholds). Only a reviewer holding all the diffs sees that.
  local pending="" n url br wt out st m_at m_epoch
  for out in "$D"/*.devin.out "$D"/*.cmd.out "$D"/*.cursor.out; do
    [ -f "$out" ] || continue
    case "$(basename "$out")" in 90-review-*|89-review-batch-*) continue;; esac
    grep -q '^exit ' "$out" 2>/dev/null || continue
    url=$(grep -oE 'https://github\.com/[^ ]+/pull/[0-9]+' "$out" 2>/dev/null | tail -1); [ -n "$url" ] || continue
    n="${url##*/}"
    review_exists "$n" && continue
    st=$(gh pr view "$n" -R Evan-Kim2028/lake-of-rage --json state,mergedAt --jq '.state+" "+(.mergedAt//"-")' 2>/dev/null)
    case "$st" in
      OPEN*) : ;;
      MERGED*) m_at="${st#MERGED }"
               [ "$m_at" = "-" ] && continue
               m_epoch=$(date -d "$m_at" +%s 2>/dev/null) || continue
               [ $(( $(date +%s) - m_epoch )) -lt 5400 ] || continue ;;
      *) continue ;;
    esac
    case " $pending " in *" $n "*) ;; *) pending="$pending $n";; esac
  done

  set -- $pending
  [ "$#" -eq 0 ] && return 0

  if [ "$#" -ge 2 ]; then
    python3 "$D/dq-mkbatchreview.py" "$(date +%H%M%S)" "$@"
    return 0
  fi

  # single uncovered PR -> per-PR review, as before
  n="$1"
  br=$(gh pr view "$n" -R Evan-Kim2028/lake-of-rage --json headRefName --jq .headRefName 2>/dev/null)
  wt=/home/evan/Documents/lake-of-rage
  for d in /home/evan/Documents/lor-wt-*; do
    [ -d "$d" ] && [ "$(git -C "$d" rev-parse --abbrev-ref HEAD 2>/dev/null)" = "$br" ] && { wt="$d"; break; }
  done
  python3 "$D/dq-mkreview.py" "$n" "$wt" "batched-scan"
}

echo "[dq] runner start maxn=$MAXN poll=${POLL}s break_after=$BREAK_AFTER"
while true; do
  bad=$(consecutive_bad)
  printf 'ts=%s live=%s queued=%s running=%s failed=%s consec_bad=%s halted=%s\n' \
    "$(date -Is)" "$(live)" "$(ls "$Q" 2>/dev/null | wc -l)" "$(ls "$R" 2>/dev/null | wc -l)" \
    "$(ls "$FAILED" 2>/dev/null | wc -l)" "$bad" "$([ -f "$HALT" ] && echo yes || echo no)" > "$HB"
  [ -f "$RL_STATE" ] && printf 'rate_limited_at=%s cooldown_s=%s eff_maxn=%s\n' \
    "$(cat "$RL_STATE")" "$COOLDOWN_S" "${eff_maxn:-$MAXN}" >> "$HB"
  if [ "$bad" -ge "$BREAK_AFTER" ] && [ ! -f "$HALT" ]; then
    echo "[dq] CIRCUIT BREAKER: $bad consecutive non-OK outcomes — halting launches. Investigate $RES then: rm $HALT" | tee -a "$D/dq.log" > "$HALT"
  fi
  # --- rate-limit backoff ---------------------------------------------------
  if [ ! -f "$RL_STATE" ] && awk -F'\t' -v since="$(cat "$TRIAL_START")" \
       '$1 > since && $6 == "RATE_LIMIT" {found=1} END{exit !found}' "$RES"; then
    date +%s > "$RL_STATE"
    echo "[dq] RATE_LIMIT at MAXN=$MAXN — pausing ${COOLDOWN_S}s, then pinning MAXN=$SAFE_MAXN" | tee -a "$D/dq.log"
  fi
  eff_maxn="$MAXN"
  if [ -f "$RL_STATE" ]; then
    rl_at=$(cat "$RL_STATE" 2>/dev/null || echo 0)
    if [ $(( $(date +%s) - rl_at )) -lt "$COOLDOWN_S" ]; then
      eff_maxn=0                      # cooling down: launch nothing
    else
      eff_maxn="$SAFE_MAXN"           # cooled off: stay at the safe cap
    fi
  fi

  if [ ! -f "$HALT" ]; then
    scan_for_reviews
    while [ "$(live)" -lt "$eff_maxn" ]; do
      nxt=$(ls "$Q"/*.md 2>/dev/null | sort | head -1); [ -n "$nxt" ] || break
      case "$(basename "$nxt")" in 89-review-batch-*|90-review-*) m=swe-2-max;; *) m=swe-2-high;; esac
      launch "$nxt" "$m" || break
      sleep 12
    done
  fi
  sleep "$POLL"
done
