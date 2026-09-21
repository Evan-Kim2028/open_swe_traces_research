#!/usr/bin/env bash
# Cross-model check: 8 certified-hard units at L0, solved by Devin swe-2-max instead of Composer.
# If Devin also fails L0 on units Composer fails at L0 and passes at L2, the flip certificate is
# model-independent rather than a Composer artifact. k=1, low concurrency (free-tier rate limits).
set -u
R=/home/evan/Documents/open_swe_traces_research
cd $R
# The CLI authenticates from ~/.devin credentials, NOT from DEVIN_API_KEY: with a fresh
# HOME it prompts for login and reports an empty model list ("Unknown model: swe-2-max").
export HOME=/home/evan
export DEVIN_API_KEY="$(uv run python -c 'from openswe_traces.pipeline.solve import devin_api_key; print(devin_api_key() or "")')"
[ -n "$DEVIN_API_KEY" ] || { echo "no devin key"; exit 1; }
harbor run --path experiments/dose_response/sweep_devin_x --agent devin --model swe-2-max \
  --n-concurrent 1 --n-attempts 1 --max-retries 1 --jobs-dir experiments/dose_response/jobs \
  --job-name sweep_devin_x2 --yes
"$R/scripts/ops/post_sweep.sh" sweep_devin_x2
