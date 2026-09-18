#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "6175bea81626118b308adf36d53287b84603cf743d7c5dcf50f3db088e11bf9a  /app/internal/locate/region_cache_test.go" | sha256sum -c --status || checksum_fail "internal/locate/region_cache_test.go"
if go test -count=1 -timeout 15m -run '^(TestRegionCache)$' ./internal/locate/...; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
