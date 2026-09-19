#!/usr/bin/env bash
# Wait for the ladder2 builder, then sync tasks to lake-vps and start the lazy driver there.
R=/home/evan/Documents/open_swe_traces_research; H=$R/experiments/harbor_nex
until [ -f $H/LADDER2.md ] && [ "$(pgrep -fc "cursor-grok-4.6""-high")" -le 2 ]; do sleep 300; done
sleep 60
KEY=$(grep -oE 'CURSOR_API_KEY=\S+' /home/evan/Documents/eval_tasks/.env | cut -d= -f2)
rsync -aq --delete $H/tasks_ladder2/ lake-vps:~/openswe/harbor_nex/tasks_ladder2/
ssh lake-vps "export CURSOR_API_KEY='$KEY'; cd ~/openswe && setsid nohup ./vps_ladder_driver.sh ~/openswe/harbor_nex/tasks_ladder2 ladder2 > ~/openswe/ladder2_driver.log 2>&1 < /dev/null &"
echo "$(date -u +%H:%M) synced and started driver on lake-vps"
