#!/usr/bin/env bash
sleep 4500
cd /home/evan/Documents/open_swe_traces_research && set -a && . /home/evan/Documents/eval_tasks/.env && set +a && exec timeout 7200 cursor-agent -p -f --model "cursor-grok-4.6""-high" --output-format text "$(cat /home/evan/devin-tasks/ablation_judge.md)"
