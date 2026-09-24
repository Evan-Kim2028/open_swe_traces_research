#!/usr/bin/env bash
# Devin swe-2-max speed probe: same fabricated unit Composer passed (store-s37-r020-L0), 1 trial.
# DEVIN_API_KEY comes from the Devin CLI credentials file; never echoed.
set -u
R=/home/evan/Documents/open_swe_traces_research
cd $R
export DEVIN_API_KEY="$(uv run python -c 'from openswe_traces.pipeline.solve import devin_api_key; print(devin_api_key() or "")')"
[ -n "$DEVIN_API_KEY" ] || { echo "no devin key"; exit 1; }
exec harbor run --path experiments/dose_response/devin_probe --agent devin --model swe-2-max \
  --n-concurrent 1 --n-attempts 1 --max-retries 1 --jobs-dir experiments/dose_response/jobs --job-name devin_probe2 --yes
