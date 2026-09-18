#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "9ad5086f61ad551dcebd21b18ebb23b3bc65678d52a29dc4bf03ca5c1138f5a9  /app/internal/unionstore/memdb_test.go" | sha256sum -c --status || checksum_fail "internal/unionstore/memdb_test.go"
echo "57e76e8b26939387373ac94fff0dce2977bc64611b79d3a9b70fe585e15acce5  /app/internal/unionstore/memdb_bench_test.go" | sha256sum -c --status || checksum_fail "internal/unionstore/memdb_bench_test.go"
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
