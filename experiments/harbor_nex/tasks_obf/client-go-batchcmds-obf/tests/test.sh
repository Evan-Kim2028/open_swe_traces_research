#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "c2a782dfc60eac2da0cddd421414cd967d8df98f6f00cc0e87fd639f60e16e9d  /app/internal/client/client_test.go" | sha256sum -c --status || checksum_fail "internal/client/client_test.go"
if go test -count=1 -timeout 15m -run '^(TestCancelTimeoutRetErr|TestBatchCommandsBuilder|TestForwardMetadataByBatchCommands)$' ./internal/client/...; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
