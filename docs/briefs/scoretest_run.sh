#!/usr/bin/env bash
R=/home/evan/Documents/open_swe_traces_research; H=$R/experiments/harbor_nex; D=/home/evan/devin-tasks
until [ -f $R/analytics/research/statefulness_vs_flip.md ]; do sleep 300; done
set -a; . /home/evan/Documents/eval_tasks/.env; set +a
cd $R && timeout 5400 cursor-agent -p -f --model "cursor-grok-4.6""-high" --output-format text "$(cat $D/openswe_score_test.md)" > $D/openswe_score_test.log 2>&1
for u in $(python3 -c "import json;print(' '.join(x['unit'] for x in json.load(open('$H/SCORETEST_PICKS.json'))))"); do
  sed "s/__UNIT__/$u/g" $D/openswe_score_unit.md > $D/openswe_score_unit_$u.md
  (cd $R && timeout 3600 cursor-agent -p -f --model "cursor-grok-4.6""-high" --output-format text "$(cat $D/openswe_score_unit_$u.md)" > $D/openswe_score_unit_$u.log 2>&1) &
done; wait
KEY=$(grep -oE 'CURSOR_API_KEY=\S+' /home/evan/Documents/eval_tasks/.env | cut -d= -f2)
rsync -aq --delete --exclude '_proof_*' $H/tasks_scoretest/ lake-vps:~/openswe/harbor_nex/tasks_scoretest/
ssh lake-vps "export CURSOR_API_KEY='$KEY'; export PATH=\$HOME/.local/bin:\$PATH; cd ~/openswe && setsid nohup harbor run --path ~/openswe/harbor_nex/tasks_scoretest --agent cursor-cli --model cursor/composer-2.5 --n-concurrent 4 --n-attempts 3 --max-retries 2 --jobs-dir ~/openswe/jobs --job-name scoretest-L2 --yes > ~/openswe/scoretest-L2.log 2>&1 < /dev/null &"
echo "$(date -u +%H:%M) scoretest launched on lake-vps"
