#!/bin/bash
# Rule A11: recompute the ns/op ceiling from gold in THIS image and rewrite test.sh.
set -euo pipefail
TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
cd /app
if [ -f "$TESTS_DIR/gold_memdb_snapshot.go" ]; then
  mkdir -p "$(dirname /app/internal/unionstore/memdb_snapshot.go)"
  cp "$TESTS_DIR/gold_memdb_snapshot.go" /app/internal/unionstore/memdb_snapshot.go
fi
if [ -d "$TESTS_DIR/hidden" ]; then
  find "$TESTS_DIR/hidden" -type f -name '*_test.go' | while read -r f; do
    rel="${f#$TESTS_DIR/hidden/}"
    mkdir -p "/app/$(dirname "$rel")"
    cp "$f" "/app/$rel"
  done
fi
BENCH_OUT=$(go test -count=1 -timeout 15m -bench='^BenchmarkSnapshotGet$' -benchtime=5000x -run='^$' ./internal/unionstore/ || true)
echo "$BENCH_OUT"
NS=$(echo "$BENCH_OUT" | awk -v n='BenchmarkSnapshotGet' '$1 ~ "^"n"(-[0-9]+)?$" {print $3; exit}')
if [ -z "${NS:-}" ]; then
  echo "benchmark BenchmarkSnapshotGet did not report ns/op" >&2
  exit 1
fi
LOAD=$(awk '{print $1}' /proc/loadavg 2>/dev/null || echo unknown)
LIMIT=$(awk -v ns="$NS" 'BEGIN { printf "%d", ns * 3 }')
echo "gold_ns=$NS load_avg=$LOAD limit=$LIMIT"
if [ -f "$TESTS_DIR/test.sh" ]; then
  sed -i "s/limit='[0-9][0-9]*'/limit='$LIMIT'/g" "$TESTS_DIR/test.sh"
fi
