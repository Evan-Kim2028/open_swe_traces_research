#!/usr/bin/env bash
# wait for the proxy-features cmd agent, then run the scaffold agent
PAT="produces outputs/proxy_""features.parquet"
while pgrep -f "$PAT" >/dev/null; do sleep 60; done
cd /home/evan/Documents/open_swe_traces_research
exec cmd -p "$(cat /home/evan/devin-tasks/openswe_scaffold.md)" -m deepseek/deepseek-v4.1-flash --effort max --yolo --skip-onboarding -t --max-turns 300
