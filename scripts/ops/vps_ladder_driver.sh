#!/usr/bin/env bash
# Lazy ladder driver on lake-vps: for each unit run A0; for fails run A1; etc. Composer 2.5 via cursor-cli. Usage: vps_ladder_driver.sh <tasks_root> <job_prefix>
set -u; export PATH="$HOME/.local/bin:$PATH"; ROOT=$1; PFX=$2; cd ~/openswe
units=$(ls $ROOT | sed -E 's/-L[0-6]$//' | sort -u)
pending="$units"
for k in 0 1 2 3 4 5 6; do
  [ -z "$pending" ] && break
  dir=~/openswe/run_${PFX}_L$k; rm -rf $dir; mkdir -p $dir
  for u in $pending; do [ -d $ROOT/$u-L$k ] && cp -r $ROOT/$u-L$k $dir/; done
  [ -z "$(ls $dir)" ] && break
  echo "$(date -u +%H:%M) level L$k units: $(ls $dir | tr '\n' ' ')"
  # rule A11: re-measure timing gates on this host (ARM) for tasks that ship measure_gold.sh
  for t in $dir/*/; do
    if [ -f $t/tests/measure_gold.sh ]; then
      img=measure-$(basename $t | tr 'A-Z' 'a-z'); docker build -q -t $img $t/environment >/dev/null 2>&1 || continue
      docker run --rm -v "$t/tests:/task/tests" $img bash -c 'cd /app && bash /task/tests/measure_gold.sh 2>&1 | tail -1' | sed "s|^|  measured $(basename $t): |"
    fi
  done
  harbor run --path $dir --agent cursor-cli --model cursor/composer-2.5 --n-concurrent 3 --n-attempts 3 --max-retries 2 --jobs-dir ~/openswe/jobs --job-name ${PFX}-L$k --yes > ~/openswe/${PFX}-L$k.log 2>&1
  pending=$(python3 - "$HOME/openswe/jobs/${PFX}-L$k/result.json" <<'PY'
import json,sys
d=json.load(open(sys.argv[1])); ev=next(iter(d['stats'].get('evals',{}).values()),{}); rs=ev.get('reward_stats',{}).get('reward',{})
import re
fails=[re.sub(r'-L\d__.*$','',t) for t in rs.get('0.0',[])]
print(' '.join(sorted(set(fails))))
PY
)
  echo "$(date -u +%H:%M) L$k done; still failing: $pending"
done
echo "$(date -u +%H:%M) driver done"
