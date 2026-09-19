#!/usr/bin/env bash
# Wait until all 4 arms have pulled metrics, then write the report via cmd.
R=/home/evan/Documents/open_swe_traces_research/experiments/curve
KG="uv run --project /home/evan/Documents/open_swe_traces_research kaggle"
arms=(random top_within_task bottom_within_task random_masked)
while true; do
  done_n=0
  for a in "${arms[@]}"; do
    slug="evandekim/openswe-curve-${a//_/-}"
    if [ ! -f "$R/arms/$a/out/metrics.json" ]; then
      s=$($KG kernels status "$slug" 2>/dev/null || true)
      if echo "$s" | grep -qE "COMPLETE|ERROR|CANCEL"; then mkdir -p "$R/arms/$a/out"; $KG kernels output "$slug" -p "$R/arms/$a/out" >/dev/null 2>&1 || true; fi
    fi
    [ -f "$R/arms/$a/out/metrics.json" ] && done_n=$((done_n+1))
  done
  echo "$(date -u +%H:%M) arms with metrics: $done_n/4"
  [ "$done_n" -eq 4 ] && break
  # also stop waiting if the queue finished and every kernel is terminal without metrics (all failed)
  sleep 900
done
cd /home/evan/Documents/open_swe_traces_research
exec timeout 7200 cmd -p "$(cat /home/evan/devin-tasks/openswe_curve_report.md)" -m deepseek/deepseek-v4.1-flash --effort max --yolo --skip-onboarding -t --max-turns 400 --output-format json
