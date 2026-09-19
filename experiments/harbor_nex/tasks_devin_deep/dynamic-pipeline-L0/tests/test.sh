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
echo "b175bf8e317331b5ae632601bbe878273158e53d46f391d342318711422d4f97  $HIDDEN/internal/unionstore/pipelined_bench_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/unionstore/pipelined_bench_test.go"
install_hidden "internal/unionstore/pipelined_bench_test.go"
echo "b175bf8e317331b5ae632601bbe878273158e53d46f391d342318711422d4f97  /app/internal/unionstore/pipelined_bench_test.go" | sha256sum -c --status || checksum_fail "internal/unionstore/pipelined_bench_test.go"
echo "24c9b4c20708b4def1fb35dce0a521e5cd20c83b19939bf8a2cb4d7cc8535682  $HIDDEN/internal/unionstore/pipelined_memdb_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/unionstore/pipelined_memdb_test.go"
install_hidden "internal/unionstore/pipelined_memdb_test.go"
echo "24c9b4c20708b4def1fb35dce0a521e5cd20c83b19939bf8a2cb4d7cc8535682  /app/internal/unionstore/pipelined_memdb_test.go" | sha256sum -c --status || checksum_fail "internal/unionstore/pipelined_memdb_test.go"
echo "5f594b703dbb153dbe2717293671a00f4b40f6e398535752c6154cad0aef0f19  $HIDDEN/internal/unionstore/pipelined_race_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/unionstore/pipelined_race_test.go"
install_hidden "internal/unionstore/pipelined_race_test.go"
echo "5f594b703dbb153dbe2717293671a00f4b40f6e398535752c6154cad0aef0f19  /app/internal/unionstore/pipelined_race_test.go" | sha256sum -c --status || checksum_fail "internal/unionstore/pipelined_race_test.go"
if go test -race -count=1 -timeout 15m -run '^(TestPipelinedFlushTrigger|TestPipelinedFlushSkip|TestPipelinedFlushBlock|TestPipelinedFlushGet|TestPipelinedFlushSize|TestPipelinedFlushGeneration|TestErrorIterator|TestPipelinedConcurrentSet)$' ./internal/unionstore/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi

BENCH_OUT=$(go test -count=1 -timeout 15m -bench='^BenchmarkPipelinedGet$' -benchtime=5000x -run='^$' ./internal/unionstore/... || true)
echo "$BENCH_OUT"
NS=$(echo "$BENCH_OUT" | awk -v n='BenchmarkPipelinedGet' '$1 ~ "^"n"(-[0-9]+)?$" {print $3; exit}')
if [ -z "$NS" ]; then
  echo "benchmark BenchmarkPipelinedGet did not report ns/op" >&2
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
if ! echo "$NS" | awk -v limit='113' '{
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
