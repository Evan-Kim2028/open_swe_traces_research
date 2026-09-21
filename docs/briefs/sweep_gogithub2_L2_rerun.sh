#!/usr/bin/env bash
# go-github L2, for real this time. The first attempt produced 30 environment-build errors
# because a docker prune had removed ladder-base:go-github; the image is rebuilt.
set -u
R=/home/evan/Documents/open_swe_traces_research
D=sweep_gogithub2_L2
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
docker image inspect ladder-base:go-github >/dev/null 2>&1 || { echo "base image missing; aborting"; exit 1; }
harbor run --path "experiments/dose_response/$D" --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 10 --n-attempts 3 --max-retries 1 --jobs-dir experiments/dose_response/jobs \
  --job-name "${D}_rerun" --yes
"$R/scripts/ops/post_sweep.sh" "${D}_rerun"
