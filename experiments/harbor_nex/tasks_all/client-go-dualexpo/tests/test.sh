#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "a67ad686e51163bd06d4f1179b7400d34b5c2a03dd0f33d3046f31d770845732  /app/config/retry/backoff_test.go" | sha256sum -c --status || checksum_fail "config/retry/backoff_test.go"
echo "cb873a2422ba3c9c3782bc880cdd3623a8bc06b91c4384e045c357f29bb4f4b6  /app/internal/client/retry/backoff_test.go" | sha256sum -c --status || checksum_fail "internal/client/retry/backoff_test.go"
if go test -count=1 -timeout 15m -run '^(TestBackoffDeepCopy)$' ./config/retry/... ./internal/client/retry/...; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
