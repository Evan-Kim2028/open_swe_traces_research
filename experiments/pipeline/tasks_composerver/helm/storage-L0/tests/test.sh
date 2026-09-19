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
echo "c4f60c950b3b7a16226f4a314ae55da12529f7401c70a5f48a8d098d03bc4686  $HIDDEN/pkg/storage/storage_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/pkg/storage/storage_bb_prop_test.go"
install_hidden "pkg/storage/storage_bb_prop_test.go"
echo "c4f60c950b3b7a16226f4a314ae55da12529f7401c70a5f48a8d098d03bc4686  /app/pkg/storage/storage_bb_prop_test.go" | sha256sum -c --status || checksum_fail "pkg/storage/storage_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestStorageContractTableProperty|TestStorageHistoryProperty|TestStorageMaxHistoryProperty|TestStorageDeployedFiltersProperty|TestStorageAdversarialProperty|TestStorageUnseenRandomProperty|TestStoragePruneFailureProperty)$' ./pkg/storage/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
