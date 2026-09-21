#!/usr/bin/env bash
# Devin as SOLVER via harbor. Harbor writes the key into credentials.toml inside the container,
# which is the correct mechanism — earlier failures were an invalid key surfacing as
# "Unknown model: 'swe-2-max'".
set -u
R=/home/evan/Documents/open_swe_traces_research
cd $R
export DEVIN_API_KEY="$(uv run python -c 'from openswe_traces.pipeline.solve import devin_api_key; print(devin_api_key() or "")')"
[ -n "$DEVIN_API_KEY" ] || { echo "no devin key"; exit 1; }
harbor run --path experiments/dose_response/devin_trial --agent devin --model swe-2-max \
  --n-concurrent 2 --n-attempts 1 --max-retries 1 --jobs-dir experiments/dose_response/jobs \
  --job-name devin_trial --yes
"$R/scripts/ops/post_sweep.sh" devin_trial
