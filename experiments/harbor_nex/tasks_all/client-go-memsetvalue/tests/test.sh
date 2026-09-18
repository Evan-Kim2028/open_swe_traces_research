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
if go test -count=1 -timeout 15m -run '^(TestOverwrite)$' ./internal/unionstore/...; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
