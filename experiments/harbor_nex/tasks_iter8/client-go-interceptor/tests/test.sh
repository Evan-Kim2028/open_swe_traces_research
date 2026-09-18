#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "c121c40c3e81e57b5c99abc7627741ce3dad9d27a67b288f85dd8f80e3657f5a  /app/tikvrpc/interceptor/interceptor_test.go" | sha256sum -c --status || checksum_fail "tikvrpc/interceptor/interceptor_test.go"
echo "1a33556c985db4d37b742ecffa2b73ddbc95b5e2bc90f0a3ae2a699c4044d6e1  /app/internal/client/client_interceptor_test.go" | sha256sum -c --status || checksum_fail "internal/client/client_interceptor_test.go"
if go test -count=1 -timeout 15m -run '^(TestInterceptedClient|TestAppendChainedInterceptor|TestInterceptor)$' ./tikvrpc/interceptor/... ./internal/client/...; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
