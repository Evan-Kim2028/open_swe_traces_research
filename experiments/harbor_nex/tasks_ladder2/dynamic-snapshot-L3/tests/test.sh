#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
HIDDEN="$TESTS_DIR/hidden"
install_hidden() {
  rel="$1"
  dest="/app/$rel"
  mkdir -p "$(dirname "$dest")"
  cp "$HIDDEN/$rel" "$dest"
}
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "48d5535721b38e869815620e77ec36a8ec49107a9c02041ef97a04808fba3737  $HIDDEN/internal/unionstore/snapshot_dyn_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/unionstore/snapshot_dyn_test.go"
install_hidden "internal/unionstore/snapshot_dyn_test.go"
echo "48d5535721b38e869815620e77ec36a8ec49107a9c02041ef97a04808fba3737  /app/internal/unionstore/snapshot_dyn_test.go" | sha256sum -c --status || checksum_fail "internal/unionstore/snapshot_dyn_test.go"
if go test -count=1 -timeout 15m -run '^(TestSnapshotStagingVisible|TestSnapshotUnmentionedKeys)$' ./internal/unionstore/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi

BENCH_OUT=$(go test -count=1 -timeout 15m -bench='^BenchmarkSnapshotGet$' -benchtime=5000x -run='^$' ./internal/unionstore/... || true)
echo "$BENCH_OUT"
NS=$(echo "$BENCH_OUT" | awk -v n='BenchmarkSnapshotGet' '$1 ~ "^"n"(-[0-9]+)?$" {print $3; exit}')
if [ -z "$NS" ]; then
  echo "benchmark BenchmarkSnapshotGet did not report ns/op" >&2
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
if ! echo "$NS" | awk -v limit='510' '{
  ns=$1+0
  if (ns > limit) {
    printf("perf gate failed: %s ns/op > %s ns/op\n", ns, limit) > "/dev/stderr"
    exit 1
  }
  printf("perf gate ok: %s ns/op <= %s ns/op\n", ns, limit)
}'; then
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
