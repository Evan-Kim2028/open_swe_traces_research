#!/usr/bin/env bash
# Resume the nex-full Harbor job after the OpenRouter free-tier daily reset (1000 req/day cap).
set -a; . /home/evan/Documents/eval_tasks/.env; set +a
R=/home/evan/Documents/open_swe_traces_research
now=$(date -u +%s); reset=1789776000; wait=$((reset-now+120)); echo "sleeping ${wait}s until reset"; sleep $((wait>0?wait:0))
cd "$R" && exec harbor jobs resume experiments/harbor_nex/jobs/nex-full --yes
