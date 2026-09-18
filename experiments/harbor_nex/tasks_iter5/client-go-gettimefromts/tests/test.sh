#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "9d530271a237723a3a41be9350d6820897d3d0e0c94db9316b455737ac9c2de3  /app/internal/latch/latch_test.go" | sha256sum -c --status || checksum_fail "internal/latch/latch_test.go"
echo "9dcacb7d31fdf4b1c76a5c0bd4eb3bf795b139e838206e28e5b244be0bd87322  /app/oracle/oracles/local_test.go" | sha256sum -c --status || checksum_fail "oracle/oracles/local_test.go"
echo "34e6ee5cadd33cc8954d01be0f8a3b47a091e829792e6623f0de2c2ea0938892  /app/oracle/oracles/pd_test.go" | sha256sum -c --status || checksum_fail "oracle/oracles/pd_test.go"
if go test -count=1 -timeout 15m -run '^(TestRecycle|TestIsExpired|TestPdOracle_GetStaleTimestamp)$' ./internal/latch/... ./oracle/oracles/...; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
