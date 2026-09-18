#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "f414b5c5cba5882038db189c55698f7605f80a88d2eb87c71192ed03a746ba89  /app/wirerpc/interceptor/interceptor_test.go" | sha256sum -c --status || checksum_fail "tikvrpc/interceptor/interceptor_test.go"
echo "f7cc0b7b5f9cff5ebc41d5e9f758d1f22d1980288920dd9f0d1fd5b393f46419  /app/internal/client/client_interceptor_test.go" | sha256sum -c --status || checksum_fail "internal/client/client_interceptor_test.go"
if go test -count=1 -timeout 15m -run '^(TestInterceptedClient|TestAppendChainedInterceptor|TestInterceptor)$' ./wirerpc/interceptor/... ./internal/client/...; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
