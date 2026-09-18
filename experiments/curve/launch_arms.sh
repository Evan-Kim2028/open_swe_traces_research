#!/usr/bin/env bash
# Push the four curve arms as separate Kaggle script kernels, then poll and pull.
#
# Kaggle script kernels take no custom env vars, so each arm gets its own directory with
# a copy of train_curve.py whose CONFIG MANIFEST default is sed'd at push time. Kaggle
# sanitizes '_' to '-' in slugs, so kernel ids are hyphenated while the arm/dir/manifest
# names keep the underscores (top_within_task -> evandekim/openswe-curve-top-within-task).
#
# Usage:
#   ./launch_arms.sh              # push all four, poll every POLL_SECONDS, pull outputs
#   DRY_RUN=1 ./launch_arms.sh    # print the commands (pushes + one poll cycle), run nothing
#
# Quota note: 30 GPU-h/week; a 2xT4 session burns 2 GPU-h per wall-clock hour. Kaggle
# caps concurrent GPU sessions (extra kernels queue), so a full 4-arm sweep may need
# more than one push round.
set -uo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
REPO="$(cd "$HERE/../.." && pwd)"
cd "$REPO"

ARMS=(random top_within_task bottom_within_task random_masked)
TIMEOUT="${TIMEOUT:-43200}"          # kernel kill-switch seconds (12 h cap)
POLL_SECONDS="${POLL_SECONDS:-300}"  # status poll interval
PUSH_ATTEMPTS="${PUSH_ATTEMPTS:-5}"
RUN=(uv run kaggle)                  # kaggle CLI from this repo's uv env
DRY_RUN="${DRY_RUN:-0}"

kernel_id() { echo "evandekim/openswe-curve-${1//_/-}"; }

run() {
  if [[ "$DRY_RUN" == "1" ]]; then
    printf '[dry] %s\n' "$*"
    return 0
  fi
  "$@"
}

prepare_arm() {
  local arm="$1" dir="$HERE/arms/$arm"
  run mkdir -p "$dir"
  run cp "$HERE/train_curve.py" "$dir/train_curve.py"
  if [[ "$DRY_RUN" == "1" ]]; then
    printf '[dry] sed -i "s/\\"MANIFEST\\": \\"random\\"/\\"MANIFEST\\": \\"%s\\"/" %s/train_curve.py\n' \
      "$arm" "$dir"
  else
    sed -i "s/\"MANIFEST\": \"random\"/\"MANIFEST\": \"$arm\"/" "$dir/train_curve.py"
  fi
}

push_arm() {
  local arm="$1" dir="$HERE/arms/$arm" attempt out rc
  for attempt in $(seq 1 "$PUSH_ATTEMPTS"); do
    if [[ "$DRY_RUN" == "1" ]]; then
      run "${RUN[@]}" kernels push -p "$dir" -t "$TIMEOUT"
      return 0
    fi
    out="$("${RUN[@]}" kernels push -p "$dir" -t "$TIMEOUT" 2>&1)"
    rc=$?
    printf '%s\n' "$out"
    if [[ $rc -eq 0 ]]; then
      return 0
    fi
    if grep -qiE '500|internal server error|service unavailable' <<<"$out"; then
      echo ">> $arm push hit a server error (attempt $attempt/$PUSH_ATTEMPTS); retrying in 60s"
      sleep 60
    else
      echo ">> $arm push failed (attempt $attempt/$PUSH_ATTEMPTS)"
      return 1
    fi
  done
  return 1
}

echo ">> Pushing ${#ARMS[@]} arms, timeout=${TIMEOUT}s per kernel"
for arm in "${ARMS[@]}"; do
  echo ">> $arm -> $(kernel_id "$arm")"
  prepare_arm "$arm"
  push_arm "$arm" || echo ">> NOTE: $arm was not pushed; rerun this script to retry it"
done

if [[ "$DRY_RUN" == "1" ]]; then
  echo ">> [dry] poll loop:"
  for arm in "${ARMS[@]}"; do
    run "${RUN[@]}" kernels status "$(kernel_id "$arm")"
  done
  for arm in "${ARMS[@]}"; do
    run mkdir -p "$HERE/arms/$arm/out"
    run "${RUN[@]}" kernels output "$(kernel_id "$arm")" -p "$HERE/arms/$arm/out"
  done
  printf '[dry] sleep %s  # repeat the poll block until every status is terminal\n' "$POLL_SECONDS"
  exit 0
fi

echo ">> Polling every ${POLL_SECONDS}s (Ctrl-C is safe; kernels keep running on Kaggle)"
declare -A pulled=()
while true; do
  all_terminal=1
  for arm in "${ARMS[@]}"; do
    kid="$(kernel_id "$arm")"
    st="$("${RUN[@]}" kernels status "$kid" 2>&1 | tr -d '\r')"
    printf '%s  %s: %s\n' "$(date +%H:%M:%S)" "$kid" "$st"
    case "$st" in
      *COMPLETE*|*ERROR*|*CANCEL_ACKNOWLEDGED*)
        if [[ -z "${pulled[$arm]:-}" ]]; then
          mkdir -p "$HERE/arms/$arm/out"
          if "${RUN[@]}" kernels output "$kid" -p "$HERE/arms/$arm/out"; then
            pulled[$arm]=1
          else
            echo ">> output pull for $arm failed; will retry on the next cycle"
          fi
        fi
        ;;
      *)
        all_terminal=0
        ;;
    esac
  done
  if [[ "$all_terminal" == "1" ]]; then
    echo ">> all arms terminal; outputs under experiments/curve/arms/<arm>/out/"
    break
  fi
  sleep "$POLL_SECONDS"
done
