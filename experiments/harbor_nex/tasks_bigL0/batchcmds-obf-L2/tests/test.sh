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
echo "75dd0640f8cbbd83994101e0365aeacaf22082a2f3d772e4637301c6f703b31d  $HIDDEN/internal/client/batch_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/client/batch_bb_prop_test.go"
install_hidden "internal/client/batch_bb_prop_test.go"
echo "75dd0640f8cbbd83994101e0365aeacaf22082a2f3d772e4637301c6f703b31d  /app/internal/client/batch_bb_prop_test.go" | sha256sum -c --status || checksum_fail "internal/client/batch_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestBatchCancelDeadlineProperty|TestBatchStreamGroupingProperty|TestBatchPackedSizesAndCancelSkip|TestBatchUnmentionedRandom)$' ./internal/client/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
