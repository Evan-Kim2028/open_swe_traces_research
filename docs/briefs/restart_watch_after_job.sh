#!/bin/bash
# When connarray k3 lands: stop solve-watch (old code), re-audit k3 with the fixed B9 rule, restart watch.
R=/home/evan/Documents/open_swe_traces_research; P=$R/experiments/pipeline
J=/home/evan/Documents/open_swe_traces_research/experiments/pipeline/jobs/client-go-connarray-L2-devin-n1-k1/
until [ -n "$(find $J -mindepth 2 -name result.json 2>/dev/null)" ]; do sleep 30; done
sleep 240   # let the in-process audit + trial row land
pat="solve""-watch --interval"; pkill -f "$pat"; sleep 3
cd $R && uv run python - <<'PY'
from pathlib import Path
from openswe_traces.pipeline.state import PipelineStore
from openswe_traces.pipeline.prepare import image_tag
from openswe_traces.pipeline_ext.hack_audit import audit_passing_attempt
P='/home/evan/Documents/open_swe_traces_research/experiments/pipeline'
s=PipelineStore(P+'/state.db')
for t in s.list_trials(include_excluded=True):
    if t.audit_class!='hacked' or t.reward!=1.0: continue
    trial=next(iter(d for d in Path(t.job_dir).iterdir() if d.is_dir()))
    task_dir=Path(P)/f'tasks/{t.repo}/{t.unit}-L{t.level}'
    v=audit_passing_attempt(image=image_tag(t.repo), trial_dir=trial, task_dir=task_dir, skip_docker=False)  # task-image seed re-run
    print(t.id, v.passed, v.hard_fails)
    if v.passed:
        s._con.execute('update trials set audit_class=?, excluded=0 where id=?', ('clean', t.id)); s._con.commit()
        s.add_event('hack_audit', f'B9 re-audit trial {t.id} {t.repo}/{t.unit}-L{t.level}: clean under scoped network rule')
PY
set -a; . /home/evan/Documents/eval_tasks/.env; set +a
nohup setsid uv run python scripts/pipeline.py solve-watch --interval 180 --host laptop > $P/logs/solve_watch_stdout.log 2>&1 &
echo "$(date -u +%H:%M) restarted solve-watch after job boundary (re-audited hacked rows)" >> $P/logs/restart_watch.log
