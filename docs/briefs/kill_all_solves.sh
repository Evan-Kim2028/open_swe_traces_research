#!/bin/bash
a="python3 scripts/pipeline.py solve"; b="-unit"
for pid in $(pgrep -f "$a$b"); do kill $pid; done
sleep 2
h="harbor ru"; n=" --path"
for pid in $(pgrep -f "$h$n"); do kill $pid; done
sleep 2
docker ps --format '{{.Names}}' | grep env-main | xargs -r docker rm -f >/dev/null 2>&1
echo "solves stopped"
