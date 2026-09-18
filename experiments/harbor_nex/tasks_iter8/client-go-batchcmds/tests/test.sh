#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "06b0994ad7aa0d5255e5683b8e0307300cfe7f758939c0913f60c6661f86ed9a  /app/internal/client/client_test.go" | sha256sum -c --status || checksum_fail "internal/client/client_test.go"
if go test -count=1 -timeout 15m -run '^(TestCancelTimeoutRetErr|TestBatchCommandsBuilder|TestForwardMetadataByBatchCommands)$' ./internal/client/...; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
