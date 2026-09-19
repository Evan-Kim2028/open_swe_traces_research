#!/bin/bash
# Run Composer verifier passes sequentially: wait for the running one, then launch the next.
D=/home/evan/devin-tasks; R=/home/evan/Documents/open_swe_traces_research
set -a; . /home/evan/Documents/eval_tasks/.env; set +a
for repo in "$@"; do
  while pgrep -f "cursor-agent.*authored/[a-z-]*/<unit>/_author" >/dev/null; do sleep 120; done
  cd "$R" && nohup timeout 10800 cursor-agent -p -f --model composer-2.5 --output-format text "$(cat $D/composer_verifier_$repo.md)" > $D/composer_verifier_$repo.log 2>&1 &
  echo "$(date -u +%H:%M) launched $repo" >> $D/composer_verifier_chain.log
  sleep 180
done
