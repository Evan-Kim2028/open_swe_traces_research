#!/usr/bin/env bash
R=/home/evan/Documents/open_swe_traces_research; A=$R/experiments/ablation_graph; D=/home/evan/devin-tasks
set -a; . /home/evan/Documents/eval_tasks/.env; set +a
rm -rf $A/repos/revive-graph/units $A/repos/revive-nograph/units
date -u > $A/repos/revive-graph/START2.txt; date -u > $A/repos/revive-nograph/START2.txt
(cd $A/repos/revive-graph && timeout 1200 cursor-agent -p -f --model "cursor-grok-4.6""-high" --output-format text "$(cat $D/ablation2_graph.md)" > $D/ablation2_graph.log 2>&1) &
(cd $A/repos/revive-nograph && env PATH="$A/shim:$PATH" timeout 1200 cursor-agent -p -f --model "cursor-grok-4.6""-high" --output-format text "$(cat $D/ablation2_nograph.md)" > $D/ablation2_nograph.log 2>&1) &
wait
cd $R && timeout 5400 cursor-agent -p -f --model "cursor-grok-4.6""-high" --output-format text "$(cat $D/ablation2_judge.md)" > $D/ablation2_judge.log 2>&1
cd $R && timeout 7200 cursor-agent -p -f --model "cursor-grok-4.6""-high" --output-format text "$(cat $D/ablation2_package.md)" > $D/ablation2_package.log 2>&1
KEY=$(grep -oE 'CURSOR_API_KEY=\S+' /home/evan/Documents/eval_tasks/.env | cut -d= -f2)
rsync -aq --delete $R/experiments/harbor_nex/tasks_ablation2/ lake-vps:~/openswe/harbor_nex/tasks_ablation2/ && ssh lake-vps "export CURSOR_API_KEY='$KEY'; cd ~/openswe && setsid nohup ./vps_ladder_driver.sh ~/openswe/harbor_nex/tasks_ablation2 abl2 > ~/openswe/abl2_driver.log 2>&1 < /dev/null &"
echo "$(date -u +%H:%M) ablation2 packaged and started on lake-vps"
