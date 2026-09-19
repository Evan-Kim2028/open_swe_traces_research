#!/usr/bin/env bash
# Detect Devin throttling across CLI sessions and Harbor devin trials; back off by stopping the newest sessions for 20 min.
D=/home/evan/devin-tasks; H=/home/evan/Documents/open_swe_traces_research/experiments/harbor_nex
PAT='429|rate.?limit|too many|capacity|throttl|quota exceeded|concurrent session'
while true; do
  hits=$(grep -liE "$PAT" $D/devin_*.log 2>/dev/null | wc -l)
  hits2=$(grep -rliE "$PAT" $H/jobs/devin-*/*/agent/*.log $H/jobs/devin-*/*/exception.txt 2>/dev/null | wc -l)
  n=$(for p in $(pgrep -f "devin ""-p"); do ps -o args= -p $p | grep -c "^devin -p"; done | grep -c 1)
  nh=$(pgrep -fc "job-name devin-")
  if [ $((hits+hits2)) -gt "${seen:-0}" ]; then
    seen=$((hits+hits2)); echo "$(date -u +%H:%M) THROTTLE seen at cli=$n harbor_jobs=$nh -> stopping newest cli session for 20 min"
    newest=$(for p in $(pgrep -f "devin ""-p"); do ps -o etimes=,pid=,args= -p $p | grep "devin -p" ; done | sort -n | head -1 | awk '{print $2}')
    [ -n "$newest" ] && { pkill -TERM -P $newest 2>/dev/null; kill $newest 2>/dev/null; }
    echo "$(date -u +%H:%M) backoff until $(date -u -d '+20 min' +%H:%M)"; sleep 1200
  fi
  sleep 120
done
