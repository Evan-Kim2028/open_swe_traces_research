#!/usr/bin/env bash
# goa L0 screen, k=1 (screening policy). First screen of this repo.
# environment/src was regenerated with scripts/ops/regen_env_src.sh after a worktree prune.
set -u
R=/home/evan/Documents/open_swe_traces_research
D="${1:-sweep_goa}"
cd $R && set -a && . /home/evan/Documents/eval_tasks/.env && set +a
docker image inspect ladder-base:goa >/dev/null 2>&1 || { echo "base image missing"; exit 1; }
harbor run --path "experiments/dose_response/$D" --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 6 --n-attempts 1 --max-retries 1 --jobs-dir experiments/dose_response/jobs --job-name "${D}_rerun" --yes
"$R/scripts/ops/post_sweep.sh" "${D}_rerun"
