#!/bin/bash
# stop a running Harbor trial by task name prefix (kills its harbor run + solve-unit child, then its containers)
u="$1"
for pid in $(pgrep -f "python3 scripts/pipeline.py solve""-unit --repo [a-z-]* --unit $u "); do kill $pid; done
for pid in $(pgrep -f "harbor run --path .*/$u-L[0-9] "); do kill $pid; done
sleep 2
docker ps --format '{{.Names}}' | grep -E "^${u,,}-l[0-9]__" | xargs -r docker rm -f >/dev/null
echo "stopped $u"
