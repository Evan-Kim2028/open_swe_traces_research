#!/usr/bin/env bash
# Parallel adversarial loop: start iteration N as soon as fewer than 3 builder agents are running. Harbor jobs (1 attempt/task) are launched by each builder.
R=/home/evan/Documents/open_swe_traces_research; D=/home/evan/devin-tasks
builders() { pgrep -fc "cursor-grok-4.6""-high"; }
echo "$(date -u +%H:%M) driver3 start"
for N in 6 7; do
  while [ "$(builders)" -ge 6 ]; do sleep 120; done   # each builder = 2 processes (timeout wrapper + agent)
  echo "$(date -u +%H:%M) === launching builder for iteration $N"
  sed "s/__N__/$N/g" $D/openswe_iter_template.md > $D/openswe_iter_$N.md
  cd $R && nohup timeout 21600 cursor-agent -p -f --model "cursor-grok-4.6""-high" --output-format text "$(cat $D/openswe_iter_$N.md)" > $D/openswe_iter_$N.log 2>&1 &
  sleep 600
done
echo "$(date -u +%H:%M) all builders launched"
