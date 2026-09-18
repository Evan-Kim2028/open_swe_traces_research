#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "33c46326c9122d24b5596bd64fb5f8840879aec80484acf18e8e6a30d60075f7  /app/internal/unionstore/memdb_test.go" | sha256sum -c --status || checksum_fail "internal/unionstore/memdb_test.go"
echo "7bfa0fd128aa45ef04fc60af4cb5825f9e560778971ef03aed27bf3167dfe00e  /app/internal/unionstore/memdb_bench_test.go" | sha256sum -c --status || checksum_fail "internal/unionstore/memdb_bench_test.go"
if go test -count=1 -timeout 15m -run '^(TestGetSet|TestKVGetSet)$' ./internal/unionstore/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi

BENCH_OUT=$(go test -count=1 -timeout 15m -bench='^BenchmarkGet$' -benchtime=5000x -run='^$' ./internal/unionstore/... || true)
echo "$BENCH_OUT"
NS=$(echo "$BENCH_OUT" | awk -v n='BenchmarkGet' '$1 ~ "^"n"(-[0-9]+)?$" {print $3; exit}')
if [ -z "$NS" ]; then
  echo "benchmark BenchmarkGet did not report ns/op" >&2
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
# awk prints the ns/op field; fail if above the gold-derived ceiling
if ! echo "$NS" | awk -v limit='754' '{
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
