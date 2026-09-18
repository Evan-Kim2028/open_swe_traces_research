#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "33c46326c9122d24b5596bd64fb5f8840879aec80484acf18e8e6a30d60075f7  /app/internal/unionstore/memdb_test.go" | sha256sum -c --status || checksum_fail "internal/unionstore/memdb_test.go"
if go test -count=1 -timeout 15m -run '^(TestOverwrite)$' ./internal/unionstore/...; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
