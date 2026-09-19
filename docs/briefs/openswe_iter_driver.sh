#!/usr/bin/env bash
# Adversarial loop driver: for N in 3..5, wait for the previous Harbor job to finish, run the builder for iteration N (which launches its own Harbor job), then wait for that job.
R=/home/evan/Documents/open_swe_traces_research; D=/home/evan/devin-tasks; J=$R/experiments/harbor_nex/jobs
finished() { python3 -c "import json,sys;d=json.load(open('$J/$1/result.json'));sys.exit(0 if d.get('finished_at') else 1)" 2>/dev/null; }
wait_job() { local name=$1 max=$2 t=0; until finished "$name"; do sleep 300; t=$((t+300)); [ $t -ge $max ] && { echo "$(date -u +%H:%M) timeout waiting for $name"; return 1; }; done; echo "$(date -u +%H:%M) $name finished"; }
exists_or_wait() { local name=$1 max=$2 t=0; until [ -f "$J/$name/result.json" ]; do sleep 300; t=$((t+300)); [ $t -ge $max ] && { echo "$(date -u +%H:%M) $name never appeared"; return 1; }; done; wait_job "$name" $((max)); }
echo "$(date -u +%H:%M) driver start"
exists_or_wait grok-xhigh-hard 21600 || true
exists_or_wait grok-xhigh-two-repo 28800 || true
for N in 3 4 5; do
  echo "$(date -u +%H:%M) === iteration $N: builder"
  sed "s/__N__/$N/g" $D/openswe_iter_template.md > $D/openswe_iter_$N.md
  cd $R && timeout 21600 cursor-agent -p -f --model "cursor-grok-4.6""-high" --output-format text "$(cat $D/openswe_iter_$N.md)" > $D/openswe_iter_$N.log 2>&1
  echo "$(date -u +%H:%M) builder $N exited rc=$?"
  exists_or_wait grok-xhigh-iter$N 36000 || true
done
echo "$(date -u +%H:%M) driver done"
