#!/bin/bash
# kill solve-unit children that are only waiting on a Devin slot (no container of theirs running); running trials are untouched
for pid in $(pgrep -f "python3 scripts/pipeline.py solve""-unit"); do
  unit=$(ps -o args= -p $pid | grep -o "\-\-unit [A-Za-z0-9_-]*" | cut -d' ' -f2)
  [ -z "$unit" ] && continue
  if docker ps --format '{{.Names}}' | grep -q "^${unit,,}-l[0-9]__"; then echo "keep $unit (running)"; else echo "kill $unit (waiting)"; kill $pid; fi
done
