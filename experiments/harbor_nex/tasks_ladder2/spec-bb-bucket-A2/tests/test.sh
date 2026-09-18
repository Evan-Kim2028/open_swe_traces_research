#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
HIDDEN="$TESTS_DIR/hidden"
install_hidden() {
  rel="$1"
  dest="/app/$rel"
  mkdir -p "$(dirname "$dest")"
  cp "$HIDDEN/$rel" "$dest"
}
checksum_fail() {
  echo 0 > /logs/verifier/reward.txt
  echo "test file modified: $1" >&2
  exit 1
}
echo "cb8e21887ef74b8f00f2905fde4e6d7e144eccac7e584be200bb5ee931670e88  $HIDDEN/internal/locate/bucket_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/locate/bucket_bb_prop_test.go"
install_hidden "internal/locate/bucket_bb_prop_test.go"
echo "cb8e21887ef74b8f00f2905fde4e6d7e144eccac7e584be200bb5ee931670e88  /app/internal/locate/bucket_bb_prop_test.go" | sha256sum -c --status || checksum_fail "internal/locate/bucket_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestBucketContainsRoundTrip|TestLocateBucketProperties|TestBucketContractExamples|TestBucketUnmentionedRandom)$' ./internal/locate/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
