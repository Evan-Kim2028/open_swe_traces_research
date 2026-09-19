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
echo "9ae06f1012ec441f58278c2c95aef27bfee5edd4c5c38167c9c72810dfffcbb2  $HIDDEN/internal/locate/regionstoresorted_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/locate/regionstoresorted_bb_prop_test.go"
install_hidden "internal/locate/regionstoresorted_bb_prop_test.go"
echo "9ae06f1012ec441f58278c2c95aef27bfee5edd4c5c38167c9c72810dfffcbb2  /app/internal/locate/regionstoresorted_bb_prop_test.go" | sha256sum -c --status || checksum_fail "internal/locate/regionstoresorted_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestSortedSearchByKeyProperty|TestSortedEndKeyProperty|TestSortedReplaceOrInsertProperty|TestRegionCacheLocateInvalidateProperty|TestRegionCacheUpdateLeaderProperty|TestSortedContractExamples|TestSortedUnmentionedRandom)$' ./internal/locate/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
