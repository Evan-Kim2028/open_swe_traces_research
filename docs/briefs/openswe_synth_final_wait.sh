#!/usr/bin/env bash
R=/home/evan/Documents/open_swe_traces_research; D=/home/evan/devin-tasks; J=$R/experiments/harbor_nex/jobs
fin() { python3 -c "import json,sys;d=json.load(open('$J/$1/result.json'));sys.exit(0 if d.get('finished_at') else 1)" 2>/dev/null; }
t=0; until [ -f $J/grok-xhigh-iter8/result.json ] && fin grok-xhigh-iter8 && [ -f $J/grok-xhigh-two-repo/result.json ] && fin grok-xhigh-two-repo; do sleep 600; t=$((t+600)); [ $t -ge 64800 ] && { echo "$(date -u +%H:%M) gave up waiting for iter8; synthesizing what exists"; break; }; done
echo "$(date -u +%H:%M) launching synthesis"
cd $R && exec timeout 14400 cursor-agent -p -f --model "cursor-grok-4.6""-high" --output-format text "$(cat $D/openswe_synth_final.md)"
