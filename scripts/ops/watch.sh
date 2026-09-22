#!/usr/bin/env bash
# One monitor pass. Prints a line ONLY when something changed or needs attention;
# silence means healthy. Re-armed every 30 minutes, so it lives in a file rather than
# being re-pasted inline each time — an alarm retyped from memory drifts, and a drifted
# alarm is how `slots.py --supervisor-pid` sat dead while looking fine.
#
#   watch.sh [iteration]   iteration 1 (or n%6==1) also prints a heartbeat
set -u
cd /home/evan/Documents/open_swe_traces_research
i="${1:-1}"
S=outputs/supervisor

# --- stability gate: report the VERDICT changing, not the drifting sample count --------
g=$(timeout 300 uv run python scripts/ops/stability_gate.py --brief 2>/dev/null)
key=$(echo "$g" | sed 's/devin-trials-sampled [0-9]* of [0-9]*//')
if [ -n "$key" ] && [ "$key" != "$(cat $S/.gate_key 2>/dev/null)" ]; then
  echo "$key" > $S/.gate_key
  case "$g" in "STABILITY PASS"*) echo "GATE RECOVERED: $g";; *) echo "GATE: $g";; esac
fi

av=$(df -BG --output=avail / | tail -1 | tr -dc '0-9')
[ -n "$av" ] && [ "$av" -lt 80 ] && echo "DISK: only ${av}G free"

sp=$(timeout 120 uv run python scripts/ops/slots.py --supervisor-pid 2>/dev/null | tr -dc '0-9')
if [ -z "$sp" ]; then echo "SUPERVISOR: not running"
elif ls -l /proc/$sp/exe 2>/dev/null | grep -q deleted; then
  echo "SUPERVISOR: pid $sp on a DELETED inode — restart it"
else
  # STALE SCRIPT. Editing supervisor.sh does nothing to a supervisor already running it:
  # bash has the loop, and only a restart picks the change up. Anything the loop invokes
  # through `uv run python` IS live, which is exactly what makes this invisible — most
  # fixes take effect immediately, so the one that does not looks like it did.
  #
  # It has now bitten twice. Five fixes sat inert for eighteen hours in one case; in the
  # other, reap_runaway --apply was wired in at 23:46 into a supervisor started at 19:08
  # and never ran once, so every runaway that night was killed by hand while the log
  # showed zero reaps. Comparing start time to file mtime catches both in one line.
  started=$(stat -c %Y /proc/$sp 2>/dev/null)
  edited=$(stat -c %Y scripts/ops/supervisor.sh 2>/dev/null)
  if [ -n "$started" ] && [ -n "$edited" ] && [ "$edited" -gt "$started" ]; then
    age=$(( (edited - started) / 60 ))
    echo "SUPERVISOR STALE: pid $sp started before supervisor.sh was last edited (${age}m earlier) — its own loop changes are INERT until restarted"
  fi
fi

dc=$(timeout 120 uv run python scripts/ops/slots.py --count 2>/dev/null | tr -dc '0-9')
dcap=$(timeout 120 uv run python scripts/ops/slots.py --cap 2>/dev/null | tr -dc '0-9')
[ -n "$dc" ] && [ -n "$dcap" ] && [ "$dc" -gt "$dcap" ] && echo "OVER-CAP: devin $dc/$dcap"

ov=$(head -1 $S/devin_cap_override 2>/dev/null)
if [ "$ov" != "$(cat $S/.cap_seen 2>/dev/null)" ]; then
  echo "$ov" > $S/.cap_seen; [ -n "$ov" ] && echo "DEVIN CAP CHANGED to $ov"; fi

# --- runaway agent commands (reap_wedged is structurally blind to these) ---------------
rw=$(timeout 300 uv run python scripts/ops/reap_runaway.py 2>/dev/null | grep -c 'WOULD KILL')
[ -n "$rw" ] && [ "$rw" -gt 0 ] && echo "RUNAWAY: $rw command(s) at full duty — reap_runaway.py --apply"

ns=$(timeout 300 uv run python -c "
import sys,glob,os; sys.path.insert(0,'scripts/ops')
import trial_guard as TG, cohorts
KNOWN={'goa-jsonrpcwire'}          # known-unrebuildable; a permanent alarm is not a signal
per=TG.ledger(); n=0
for d in glob.glob('experiments/dose_response/sweep_*/*/'):
    b=os.path.basename(d.rstrip('/'))
    if '-L' not in b: continue
    base,suf=b.rsplit('-L',1)
    if base in KNOWN or not suf[:1].isdigit(): continue
    if not glob.glob(d+'tests/hidden/**/*',recursive=True): continue
    if cohorts.barren(d.split(os.sep)[2]): continue
    if os.path.isdir(os.path.join(d,'environment','src')): continue
    # PER SOLVER. Asked with the merged guard this printed 2 while 123 units were runnable
    # for devin or grok and blocked on a missing tree -- a 60x undercount, because the cap
    # went per solver and the merged view answers a question nobody asks any more. The
    # number matters: orchestrate SKIPS a no-src unit rather than failing it, so this
    # backlog is silent by construction, and it is free-solver work idling while devin is
    # the bottleneck and composer is the paid one.
    for sv in ('composer','devin','grok'):
        try: ok,_=TG.decide(b,per,solver=sv)
        except Exception: ok=False
        if ok: n+=1; break
print(n)" 2>/dev/null | tr -dc '0-9')
[ -n "$ns" ] && [ "$ns" -gt 0 ] && echo "MISSING-SRC: $ns unit(s) runnable for some solver have no environment/src — scripts/ops/restore_env_src.py"

ss=$(timeout 300 uv run python scripts/ops/second_screen.py --report 2>/dev/null | sed -n '1,3p;/by repo/,+4p' | tr '\n' ' ')
if [ -n "$ss" ] && [ "$ss" != "$(cat $S/.ss_seen 2>/dev/null)" ]; then
  echo "$ss" > $S/.ss_seen; echo "SECOND SCREEN: $ss"; fi

# --- budget: the STATE CHANGE, not the state (SHORT is persistent and would spam) ------
bf=$(timeout 600 uv run python scripts/ops/budget_forecast.py --brief 2>/dev/null)
bstate=$(echo "$bf" | awk '{print $1}')
if [ -n "$bstate" ] && [ "$bstate" != "$(cat $S/.budget_state 2>/dev/null)" ]; then
  echo "$bstate" > $S/.budget_state; echo "BUDGET STATE -> $bf"; fi
left=$(echo "$bf" | sed -n 's/.*budget \([0-9]*\)M.*/\1/p')
if [ -n "$left" ] && [ "$left" -le 5 ] && [ "$(cat $S/.budget_exhausted 2>/dev/null)" != yes ]; then
  echo yes > $S/.budget_exhausted
  echo "BUDGET EXHAUSTED: ${left}M left — gate refusing cohorts"; fi

cv=$(timeout 300 uv run python -c "
import sys; sys.path.insert(0,'scripts/ops')
import trial_ledger as TL
print(sum(1 for d in TL.certificates_by_solver().values() if len(d)>1))" 2>/dev/null | tr -dc '0-9')
if [ -n "$cv" ] && [ "$cv" != "$(cat $S/.curves_seen 2>/dev/null)" ]; then
  echo "$cv" > $S/.curves_seen
  [ "$cv" -gt 0 ] && echo "TWO-CURVE COVERAGE: $cv unit(s) have independent curves from both solvers"; fi

if [ $((i % 6)) -eq 1 ]; then
  d=$(timeout 300 uv run python -c "
import sys; sys.path.insert(0,'scripts/ops')
import trial_ledger as TL
from collections import Counter
c=TL.certificates(); s=TL.summary()
multi=sum(1 for v in c.values() if len(v.get('solvers',[]))>1)
print(f\"certs {len(c)} multi-solver {multi} | {s['tokens']/1e9:.2f}B \${s['cost_usd']:.2f}\")" 2>/dev/null)
  echo "HEARTBEAT: $d | $bf | devin ${dc}/${dcap} | disk ${av}G | no-src ${ns} | runaway ${rw} | curves ${cv}"
fi
