#!/usr/bin/env bash
# Continuous operator. Keeps the pipeline fed without a human in the loop, and records a
# rolling window of what actually changed so a flat bank is visible as a stall.
#
# Every TICK it: refills Devin slots from the manifest, launches the next queued sweep when
# trial concurrency allows, reaps only provably-wedged containers, prunes only under pressure,
# and appends one snapshot to outputs/rolling.jsonl.
#
#   supervisor.sh [tick_sec]        foreground loop (run it with nohup)
set -u
R=/home/evan/Documents/open_swe_traces_research
cd "$R"
TICK="${1:-300}"
CAP=12                 # trial containers; above this docker's address pool starts failing
DEVIN_SLOTS=4          # user call: run at 4 and absorb the occasional throttle, rather than throttling ourselves to 2
QUEUE="$R/outputs/supervisor/sweep_queue.txt"
LEDGER="$R/outputs/rolling.jsonl"
LOG="$R/outputs/supervisor/supervisor.log"
touch "$QUEUE"

say() { echo "[$(date +%H:%M:%S)] $*" | tee -a "$LOG"; }

trial_containers() { docker ps --format '{{.Names}}' 2>/dev/null | grep -c 'env-main' || true; }
devin_up()  { pgrep -af '[d]evin --model' 2>/dev/null | grep -oP 'closure_\w+' | sort -u; }
# Devin trials run the CLI INSIDE a container, so they never appear in the host process
# table - but they spend the same account's request quota. Counting only host sessions
# under-reports concurrency and would let trials silently blow the rate limit.
devin_trials() {
  # Count harbor runs whose --agent is devin, multiplied by their concurrency. The
  # container itself does not expose DEVIN_API_KEY in Config.Env, so inspecting env vars
  # returned zero; the launching harbor process is the reliable marker.
  # Dedupe by job name: "timeout N harbor run --job-name X" and its child "harbor run
  # --job-name X" both match, which double-counted a single trial as two.
  ps -eo args 2>/dev/null | awk '
    /harbor run/ && !/awk/ {
      agent=""; conc=1
      for (i=1;i<NF;i++) {
        if ($i=="--agent") agent=$(i+1)
        if ($i=="--n-concurrent") conc=$(i+1)
      }
      job=""
      for (i=1;i<NF;i++) if ($i=="--job-name") job=$(i+1)
      if (agent=="devin" && !(job in seen)) { seen[job]=1; total+=conc }
    }
    END { print total+0 }'
}
devin_load() { echo $(( $(devin_up | wc -l) + $(devin_trials) )); }
# The sweep NAME is the script's first argument. Grepping for any sweep_* token also
# matches "sweep_seq" — the script's own filename — inflating the count by one and
# blocking new launches while container slots sat idle.
sweeps_up() {
  ps -eo args 2>/dev/null \
    | awk '/sweep_seq\.sh/ && !/awk/ {for(i=1;i<NF;i++) if($i ~ /sweep_seq\.sh$/) {print $(i+1); break}}' \
    | grep -v '^$' | sort -u
}

# --- refill Devin from the manifest ------------------------------------------------
# Two failures this refill has to avoid. It once launched three sessions in 40s against a
# 2-slot cap, because a freshly launched session is not visible to pgrep before the next
# iteration re-counts. And when the free-tier limit is active, every launch dies on arrival,
# so refilling just burns the quota we are trying to protect.
LAST_LAUNCH=0
COOLDOWN=150           # seconds between launches, so a new session becomes visible first
THROTTLE_BACKOFF=1800  # 30m. The limit reported "resets in 6 minutes" but refilling at 15m still died, so wait it out properly.

# The limit is cumulative request volume over a rolling window, not concurrency: 09-19 and
# 09-20 had the same session count and active minutes, but today carried 87% more messages
# and 88% more tool calls because the briefs now verify every unit in Docker. Dropping slots
# would cut throughput without raising the ceiling.
#
# So probe rather than wait out a fixed timer. A manual probe survived at 21 minutes while a
# flat 30-minute backoff was still blocking refills, wasting 9 minutes of every cycle. After
# a short floor, let one session try: if the quota is still exhausted it dies in seconds and
# costs nothing, and refill_devin's own cooldown stops it becoming a hot loop.
# Seconds since the most recent throttle message, or a large number if none.
throttle_age_sec() {
  local newest
  newest=$(grep -lE "Reached free model rate limit|Your limit will reset in" \
             /home/evan/Documents/oswt-*/outputs/*.log 2>/dev/null \
           | xargs -r stat -c %Y 2>/dev/null | sort -rn | head -1)
  if [ -z "${newest:-}" ]; then echo 999999; else echo $(( $(date +%s) - newest )); fi
}

refill_devin() {
  local up n job brief wt now
  # Run 4. On throttle: 30m cooloff at 0, then 2 slots for an hour, then back to 4.
  # No predictive budget: the estimated accumulator said "1 slot" while four sessions ran
  # productively for 50 minutes with no throttle in 71, because tool-call timestamps are
  # approximated and the cap rested on two events measured the same crude way.
  # Reserve half the Devin budget for TRIALS. Authoring sessions were filling all four
  # slots, so the supervisor correctly refused to start any trial and the bank stopped
  # moving: Composer was draining to zero at the same time. Sessions cap at 2, leaving 2
  # for trials, which are the only thing that certifies a unit.
  local want thr_age session_cap=2
  thr_age=$(throttle_age_sec)
  if [ "$thr_age" -lt 1800 ]; then
    say "throttled ${thr_age}s ago — 30m cooloff, not refilling"
    return 0
  elif [ "$thr_age" -lt 5400 ]; then
    want=1                       # past the 30m cooloff, reduced for the next hour
  else
    want=$session_cap
  fi
  # Sessions are capped separately; devin_load still bounds the account total.
  up=$(devin_up | wc -l)
  [ "$up" -ge "$want" ] && return 0
  [ "$(devin_load)" -ge 4 ] && { say "devin account at $(devin_load)/4 — no refill"; return 0; }
  while IFS=$'\t' read -r job brief wt branch model tmo deliv _; do
    [ -z "${job:-}" ] && continue
    case "$job" in \#*) continue;; esac
    now=$(date +%s)
    [ $(( now - LAST_LAUNCH )) -lt "$COOLDOWN" ] && break
    up=$(devin_up | wc -l); [ "$up" -ge "$DEVIN_SLOTS" ] && break
    devin_up | grep -qx "closure_$job" && continue
    [ -n "${deliv:-}" ] && [ "$deliv" != "-" ] && [ -f "$wt/$deliv" ] && continue
    [ -f "$brief" ] || continue
    [ -d "$wt" ] || continue
    mkdir -p "$wt/outputs"
    say "devin refill: $job"
    ( cd "$wt" && setsid nohup timeout "${tmo:-14400}" \
        devin --model "${model:-swe-2-max}" --permission-mode dangerous -p \
        --prompt-file "$brief" > "outputs/$job.log" 2>&1 < /dev/null & )
    LAST_LAUNCH=$(date +%s)
    sleep 25
    break          # one launch per tick; the next tick re-evaluates with real counts
  done < /home/evan/devin-tasks/queue/manifest.tsv
}

# --- launch the next queued sweep when there is room --------------------------------
pump_sweeps() {
  local c s next
  c=$(trial_containers)
  [ "$c" -ge "$CAP" ] && { say "trials at cap ($c/$CAP)"; return 0; }
  # Previously: one sweep at a time. That let a nearly-finished sweep holding one container
  # block the other eleven slots — measured a full 30m window at 1/12 utilisation and a flat
  # bank. Gate on spare CAPACITY instead, which is the resource that actually runs out.
  local nsweeps headroom
  nsweeps=$(sweeps_up | wc -l)
  headroom=$(( CAP - c ))
  [ "$headroom" -lt 4 ] && { say "only ${headroom} container slot(s) free — not starting another sweep"; return 0; }
  # Containers are the scarce resource (CPU-bound at ~12), not sweep processes. A cap of 3
  # sweeps throttled launches while three winding-down sweeps held only 3 containers between
  # them and 9 slots sat idle. Keep a ceiling to bound docker network use, but a loose one.
  [ "$nsweeps" -ge 5 ] && { say "5 sweeps already running"; return 0; }
  next=$(grep -vE '^\s*(#|$)' "$QUEUE" | head -1) || true
  [ -z "${next:-}" ] && return 0
  [ -d "experiments/dose_response/$next" ] || { say "queue: $next missing, dropping"; sed -i "0,/^$next$/{/^$next$/d}" "$QUEUE"; return 0; }
  # Do not launch a cohort that is still being written. sweep_kops3_L2 was queued while
  # STAGE3 was mid-write; the sweep saw a half-staged tree, the gates dropped everything
  # and it reported "nothing left to trial" on 17 perfectly good units.
  # -mmin, not -newermt with a relative date: find here is bfs, which rejects
  # "-2 minutes" as a date. With stderr swallowed that read as "nothing written
  # recently", so this guard silently never fired.
  newest=$(find "experiments/dose_response/$next" -type f -mmin -2 2>/dev/null | head -1)
  if [ -n "$newest" ]; then say "queue: $next still being written, waiting"; return 0; fi
  # Split by rung, measured rather than guessed. A Devin trial takes ~19 min against
  # Composer's ~15, but Devin allows 1-2 concurrent against Composer's 12, so Devin is
  # 7-13x slower in throughput while being free. L2 is where certification happens and the
  # volume is low, so it goes to Devin; L0 screening is high-volume and cheap per unit, so
  # it stays on Composer where throughput matters.
  # Devin trials share the 4-slot account cap with authoring sessions, so concurrency is
  # whatever headroom is left, and we do not launch at all without at least one free slot.
  local room rounds
  case "$next" in
    *_L2)
      room=$(( 4 - $(devin_load) ))
      if [ "$room" -lt 1 ]; then
        say "no devin headroom ($(devin_load)/4) — holding $next"
        return 0
      fi
      [ "$room" -gt 2 ] && room=2
      say "sweep launch: $next on DEVIN (free, concurrency $room, 2 rounds)"
      sed -i "0,/^$next$/{/^$next$/d}" "$QUEUE"
      AGENT=devin nohup bash ./scripts/ops/sweep_seq.sh "$next" 2 "$room" \
        > "outputs/${next}.log" 2>&1 &
      ;;
    *)
      # L0 screen on Composer: one round, because at L0 a FAILURE is the verdict we want
      # and retrying it buys nothing. Composer is capped at 250M tokens from the baseline
      # set on 2026-09-20; Devin is free and unbounded, so when the cap is gone L0 goes to
      # Devin too rather than stopping.
      if ! uv run python scripts/ops/composer_budget.py --quiet >/dev/null 2>&1; then
        room=$(( 4 - $(devin_load) ))
        if [ "$room" -lt 1 ]; then say "composer budget spent, no devin headroom — holding $next"; return 0; fi
        [ "$room" -gt 2 ] && room=2
        say "composer budget SPENT — running $next on devin instead (concurrency $room)"
        sed -i "0,/^$next$/{/^$next$/d}" "$QUEUE"
        AGENT=devin nohup bash ./scripts/ops/sweep_seq.sh "$next" 1 "$room" \
          > "outputs/${next}.log" 2>&1 &
        return 0
      fi
      room=$(( CAP - $(trial_containers) ))
      [ "$room" -gt 8 ] && room=8
      say "sweep launch: $next on COMPOSER (concurrency $room, 1 round)"
      sed -i "0,/^$next$/{/^$next$/d}" "$QUEUE"
      nohup bash ./scripts/ops/sweep_seq.sh "$next" 1 "$room" \
        > "outputs/${next}.log" 2>&1 &
      ;;
  esac
}

# --- rolling window ------------------------------------------------------------------
snapshot() {
  uv run python - <<'PY' >> "$LEDGER" 2>/dev/null || true
import json,time,subprocess,sys
sys.path.insert(0,"scripts/ops")
try:
    import trial_guard as g
    per=g.ledger()
    cert=[u for u,d in per.items() if d.get("0") and max(d["0"])==0 and d.get("2") and max(d["2"])>0]
    easy=[u for u,d in per.items() if d.get("0") and max(d["0"])>0]
    nf=[u for u,d in per.items() if d.get("0") and max(d["0"])==0
        and len(d.get("2",[]))>=g.NONFLIP_CAP and max(d.get("2") or [1])==0]
    trials=sum(len(v) for d in per.values() for v in d.values())
except Exception as e:
    per,cert,easy,nf,trials={},[],[],[],0
def sh(c):
    try: return subprocess.run(c,shell=True,capture_output=True,text=True,timeout=20).stdout.strip()
    except Exception: return ""
print(json.dumps({
  "t": int(time.time()),
  "iso": time.strftime("%Y-%m-%dT%H:%M:%S"),
  "certified": len(cert), "too_easy": len(easy), "nonflip": len(nf),
  "units": len(per), "trials": trials,
  "devin": sh("pgrep -af '[d]evin --model' | grep -oP 'closure_\\\\w+' | sort -u | tr '\\\\n' ' '"),
  "sweeps": sh("pgrep -af '[s]weep_seq' | grep -oP 'sweep_[a-zA-Z0-9_]+' | grep -v '^sweep_seq$' | sort -u | tr '\\\\n' ' '"),
  "containers": int(sh("docker ps --format '{{.Names}}' | grep -c env-main") or 0),
  "free_gb": int((sh("df --output=avail -BG / | tail -1") or "0G").rstrip("G ") or 0),
}))
PY
}

say "supervisor start: tick=${TICK}s cap=${CAP} devin_slots=${DEVIN_SLOTS}"
while true; do
  # Keep the queue fed: author -> verify -> stage -> trial, one new job per tick, newest
  # stage first. Without this every stage stalled whenever nobody noticed it had finished.
  # Harvest BEFORE autogen. A VF/RC job writes its output inside its own worktree and
  # nothing moved it to the main checkout, so 74 finished units were invisible to the
  # stager while autogen queued more authoring behind them. Harvest first so autogen
  # and orchestrate both see the true state.
  # Enforce the Devin cap every tick, before anything else decides to start work.
  # Two independent launchers (dq2 sessions, orchestrate trials) each stayed under
  # the cap by their own count while the total ran to 5, 6, 7. This trims trials,
  # never sessions.
  uv run python scripts/ops/devin_cap.py --apply >> "$LOG" 2>&1 || true
  # Track Grok/Cursor jobs too. The loop watched Devin and containers only, so a
  # stalled authoring or verification session was invisible until someone looked.
  uv run python scripts/ops/jobs_status.py >> "$LOG" 2>&1 || true
  # The stability gate is the current objective: a clean 6h window is what turns
  # scaling into a pure budget decision. Logged every tick so progress is visible
  # without anyone asking for it.
  uv run python scripts/ops/stability_gate.py >> "$LOG" 2>&1 || true
  uv run python scripts/ops/harvest.py --apply >> "$LOG" 2>&1 || true
  uv run python scripts/ops/pipeline_autogen.py >> "$LOG" 2>&1 || true
  # Devin session launching belongs to /home/evan/devin-tasks/dq2.sh, which predates this
  # supervisor and has its own MAXN, cooldown and single-instance guard. Both launchers
  # reading the same manifest is why sessions I killed kept reappearing and why the count
  # sat at 4 against a cap of 2. One launcher only; this one keeps sweeps, autogen, health.
  # refill_devin   # disabled: dq2.sh owns Devin sessions
  # Sweep launching belongs to orchestrate.py. Running pump_sweeps as well would recreate
  # the two-launcher bug that made killed sessions reappear and the count sit at 4 vs a
  # cap of 2. One owner per resource.
  uv run python scripts/ops/orchestrate.py --apply >> "$LOG" 2>&1 || true
  # reap only what is provably dead; the dry-run/real distinction is inside the script
  bash scripts/ops/reap_wedged.sh 45 45 >> "$LOG" 2>&1 || true
  free=$(df --output=avail -BG / | tail -1 | tr -dc '0-9')
  if [ "${free:-999}" -lt 100 ]; then say "disk ${free}G < 100G, pruning"; bash scripts/ops/prune_worktrees.sh >> "$LOG" 2>&1 || true; fi
  uv run python scripts/ops/devin_ratelimit_check.py >> "$LOG" 2>&1 || say "RATE LIMIT SIGNATURE — see supervisor.log"
  snapshot
  uv run python scripts/ops/build_dashboard.py >> "$LOG" 2>&1 || true
  sleep "$TICK"
done
