#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
mkdir -p /logs/artifacts
if [ -d /pristine ]; then
  ( cd / && if command -v git >/dev/null 2>&1; then git diff --no-index --no-color pristine app; else diff -ruN pristine app; fi )     | sed -e 's|a/pristine/|a/|g' -e 's|b/app/|b/|g' -e 's|a/app/|a/|g' -e 's|b/pristine/|b/|g' -e 's|^--- pristine/|--- a/|' -e 's|^+++ app/|+++ b/|'     > /logs/artifacts/agent.patch || true
fi
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
echo "893a5e157a52ca03844d1488b91ea6dd06db6a96f0a36171ab464b25929ec85f  $HIDDEN/grpc/middleware/grpctrace_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/grpc/middleware/grpctrace_bb_prop_test.go"
install_hidden "grpc/middleware/grpctrace_bb_prop_test.go"
echo "893a5e157a52ca03844d1488b91ea6dd06db6a96f0a36171ab464b25929ec85f  /app/grpc/middleware/grpctrace_bb_prop_test.go" | sha256sum -c --status || checksum_fail "grpc/middleware/grpctrace_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestGrpctraceUnaryContinueTrace|TestGrpctraceUnaryNewTrace|TestGrpctraceUnaryDiscardRandom|TestGrpctraceZeroRateProperty|TestGrpctraceZeroRateRandom|TestGrpctraceStreamServerContext|TestGrpctraceStreamContextRandom|TestGrpctraceUnaryClientMetadata|TestGrpctraceClientMetadataRandom|TestGrpctraceClientNoTrace|TestGrpctraceClientNoTraceRandom|TestGrpctraceStreamContinueTraceRandom|TestGrpctraceUnseenRandomProperty)$' ./grpc/middleware/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
