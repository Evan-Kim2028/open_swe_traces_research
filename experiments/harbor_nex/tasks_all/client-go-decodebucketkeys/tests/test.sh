#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "a1bcc096cec8658aee9f5a9c782ed8972c69d4d12e740ee581b7835d588cd14e  /app/internal/apicodec/codec_v2_test.go" | sha256sum -c --status || checksum_fail "internal/apicodec/codec_v2_test.go"
if go test -count=1 -timeout 15m -run '^(TestCodecV2)$' ./internal/apicodec/...; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
