#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "9dcacb7d31fdf4b1c76a5c0bd4eb3bf795b139e838206e28e5b244be0bd87322  /app/oracle/oracles/local_test.go" | sha256sum -c --status || checksum_fail "oracle/oracles/local_test.go"
if go test -count=1 -timeout 15m -run '^(TestLocalOracle)$' ./oracle/oracles/...; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
