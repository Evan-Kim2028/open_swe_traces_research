#!/bin/bash
# stop every pipeline process on the laptop; patterns are built from split strings so this script never matches itself
p1="pipeline.py sol"; p2="ve-"
h1="harbor ru"; h2="n --path"
for pat in "$p1$p2" "$h1$h2" "docker_cleanup""_loop" "devin_throttle""_watch" "agent_""watchdog" "composer_verifier""_chain" "restart_watch""_"; do
  for pid in $(pgrep -f "$pat"); do [ "$pid" != "$$" ] && kill "$pid" 2>/dev/null; done
done
sleep 3
docker ps -q | xargs -r docker rm -f >/dev/null 2>&1
echo "remaining: harbor=$(pgrep -fc "$h1$h2") watch=$(pgrep -fc "$p1$p2") containers=$(docker ps -q | wc -l)"
