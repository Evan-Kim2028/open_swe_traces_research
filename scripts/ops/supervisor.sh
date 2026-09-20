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
DEVIN_SLOTS=2          # free model tier throttled at 4-5 concurrent on 2026-09-20 18:2x
QUEUE="$R/outputs/supervisor/sweep_queue.txt"
LEDGER="$R/outputs/rolling.jsonl"
LOG="$R/outputs/supervisor/supervisor.log"
touch "$QUEUE"

say() { echo "[$(date +%H:%M:%S)] $*" | tee -a "$LOG"; }

trial_containers() { docker ps --format '{{.Names}}' 2>/dev/null | grep -c 'env-main' || true; }
devin_up()  { pgrep -af '[d]evin --model' 2>/dev/null | grep -oP 'closure_\w+' | sort -u; }
sweeps_up() { pgrep -af '[s]weep_seq' 2>/dev/null | grep -oP 'sweep_[a-zA-Z0-9_]+' | sort -u; }

# --- refill Devin from the manifest ------------------------------------------------
refill_devin() {
  local up n job brief wt
  up=$(devin_up | wc -l)
  [ "$up" -ge "$DEVIN_SLOTS" ] && return 0
  while IFS=$'\t' read -r job brief wt branch model tmo deliv _; do
    [ -z "${job:-}" ] && continue
    case "$job" in \#*) continue;; esac
    up=$(devin_up | wc -l); [ "$up" -ge "$DEVIN_SLOTS" ] && break
    devin_up | grep -qx "closure_$job" && continue
    # a job whose deliverable already exists is done, not dead
    [ -n "${deliv:-}" ] && [ "$deliv" != "-" ] && [ -f "$wt/$deliv" ] && continue
    [ -f "$brief" ] || continue
    [ -d "$wt" ] || continue
    mkdir -p "$wt/outputs"
    say "devin refill: $job"
    ( cd "$wt" && setsid nohup timeout "${tmo:-14400}" \
        devin --model "${model:-swe-2-max}" --permission-mode dangerous -p \
        --prompt-file "$brief" > "outputs/$job.log" 2>&1 < /dev/null & )
    sleep 20
  done < /home/evan/devin-tasks/queue/manifest.tsv
}

# --- launch the next queued sweep when there is room --------------------------------
pump_sweeps() {
  local c s next
  c=$(trial_containers)
  [ "$c" -ge "$CAP" ] && { say "trials at cap ($c/$CAP)"; return 0; }
  [ -n "$(sweeps_up)" ] && return 0          # one sweep at a time; it manages its own rounds
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
  "sweeps": sh("pgrep -af '[s]weep_seq' | grep -oP 'sweep_[a-zA-Z0-9_]+' | sort -u | tr '\\\\n' ' '"),
  "containers": int(sh("docker ps --format '{{.Names}}' | grep -c env-main") or 0),
  "free_gb": int((sh("df --output=avail -BG / | tail -1") or "0G").rstrip("G ") or 0),
}))
PY
}

say "supervisor start: tick=${TICK}s cap=${CAP} devin_slots=${DEVIN_SLOTS}"
while true; do
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
