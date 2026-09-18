#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "dc152e71047f83827c951fae858a1739d2feed26bc2b8207ec6ab754c26865be  /app/internal/locate/region_cache_test.go" | sha256sum -c --status || checksum_fail "internal/locate/region_cache_test.go"
echo "7c380e9ea62815028bf84ccd09a35960f6b5abf514e4fc01aaff89923455b684  /app/internal/apicodec/codec_v2_test.go" | sha256sum -c --status || checksum_fail "internal/apicodec/codec_v2_test.go"
if go test -count=1 -timeout 15m -run '^(TestCodecV2|TestRegionCache)$' ./internal/locate/... ./internal/apicodec/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
