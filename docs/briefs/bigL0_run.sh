#!/usr/bin/env bash
R=/home/evan/Documents/open_swe_traces_research; H=$R/experiments/harbor_nex; D=/home/evan/devin-tasks
set -a; . /home/evan/Documents/eval_tasks/.env; set +a
cd $R && timeout 7200 cursor-agent -p -f --model "cursor-grok-4.6""-high" --output-format text "$(cat $D/openswe_bigL0_author.md)" > $D/openswe_bigL0_author.log 2>&1
cd $R && timeout 7200 cursor-agent -p -f --model "cursor-grok-4.6""-high" --output-format text "$(cat $D/openswe_bigL0_verifier.md)" > $D/openswe_bigL0_verifier.log 2>&1
mkdir -p $H/tasks_bigL0_run && rm -rf $H/tasks_bigL0_run/* && cp -r $H/tasks_bigL0/*-L0 $H/tasks_bigL0/*-L2 $H/tasks_bigL0_run/ 2>/dev/null
cd $R && setsid nohup harbor run --path experiments/harbor_nex/tasks_bigL0_run --agent cursor-cli --model cursor/composer-2.5 --n-concurrent 2 --n-attempts 3 --max-retries 2 --jobs-dir experiments/harbor_nex/jobs --job-name bigL0 --yes > experiments/harbor_nex/bigL0.log 2>&1 < /dev/null &
echo "$(date -u +%H:%M) bigL0 launched"
