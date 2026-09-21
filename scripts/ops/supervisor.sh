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

throttled_recently() {
  local newest
  newest=$(grep -lE "Reached free model rate limit|Your limit will reset in" \
             /home/evan/Documents/oswt-*/outputs/*.log 2>/dev/null \
           | xargs -r stat -c %Y 2>/dev/null | sort -rn | head -1)
  [ -z "${newest:-}" ] && return 1
  [ $(( $(date +%s) - newest )) -lt "$THROTTLE_BACKOFF" ]
}

refill_devin() {
  local up n job brief wt now
  if throttled_recently; then
    say "devin: throttle signature within ${THROTTLE_BACKOFF}s — not refilling"
    return 0
  fi
  up=$(devin_up | wc -l)
  [ "$up" -ge "$DEVIN_SLOTS" ] && return 0
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
  say "sweep launch: $next"
  sed -i "0,/^$next$/{/^$next$/d}" "$QUEUE"
  nohup bash ./scripts/ops/sweep_seq.sh "$next" 3 8 > "outputs/${next}.log" 2>&1 &
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
  uv run python scripts/ops/pipeline_autogen.py >> "$LOG" 2>&1 || true
  refill_devin
  pump_sweeps
  # reap only what is provably dead; the dry-run/real distinction is inside the script
  bash scripts/ops/reap_wedged.sh 45 45 >> "$LOG" 2>&1 || true
  free=$(df --output=avail -BG / | tail -1 | tr -dc '0-9')
  if [ "${free:-999}" -lt 100 ]; then say "disk ${free}G < 100G, pruning"; bash scripts/ops/prune_worktrees.sh >> "$LOG" 2>&1 || true; fi
  uv run python scripts/ops/devin_ratelimit_check.py >> "$LOG" 2>&1 || say "RATE LIMIT SIGNATURE — see supervisor.log"
  snapshot
  uv run python scripts/ops/build_dashboard.py >> "$LOG" 2>&1 || true
  sleep "$TICK"
done
