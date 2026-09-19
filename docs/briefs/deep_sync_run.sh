#!/usr/bin/env bash
R=/home/evan/Documents/open_swe_traces_research; H=$R/experiments/harbor_nex
until [ -f $H/LADDER_DEEP.md ]; do sleep 300; done; sleep 120
H2=$H/tasks_deep; python3 - "$H2" <<'PY'
import os,re,sys
root=sys.argv[1]
for d in sorted(os.listdir(root)):
    m=re.fullmatch(r"(.+)-A(-?[0-4])",d)
    if m: os.rename(os.path.join(root,d),os.path.join(root,f"{m.group(1)}-L{int(m.group(2))+2}"))
PY
KEY=$(grep -oE 'CURSOR_API_KEY=\S+' /home/evan/Documents/eval_tasks/.env | cut -d= -f2)
rsync -aq --delete --exclude '_proof_*' --exclude '_gold_src' $H/tasks_deep/ lake-vps:~/openswe/harbor_nex/tasks_deep/
ssh lake-vps "export CURSOR_API_KEY='$KEY'; export PATH=\$HOME/.local/bin:\$PATH; cd ~/openswe && setsid nohup ./vps_ladder_driver.sh ~/openswe/harbor_nex/tasks_deep deep > ~/openswe/deep_driver.log 2>&1 < /dev/null &"
echo "$(date -u +%H:%M) deep synced and started"
