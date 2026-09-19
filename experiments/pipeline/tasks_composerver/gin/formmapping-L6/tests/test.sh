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
echo "ba75214aed51e8465c7faecdfa055183651f1051fd8f3757738bbb1ad10f804b  $HIDDEN/binding/formmapping_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/binding/formmapping_bb_prop_test.go"
install_hidden "binding/formmapping_bb_prop_test.go"
echo "ba75214aed51e8465c7faecdfa055183651f1051fd8f3757738bbb1ad10f804b  /app/binding/formmapping_bb_prop_test.go" | sha256sum -c --status || checksum_fail "binding/formmapping_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestFMContractTableProperty|TestFMCollectionFormatProperty|TestFMTimeDurationProperty|TestFMStructMapProperty|TestFMPtrCircularProperty|TestFMCustomUnmarshalProperty|TestFMBindersProperty|TestFMMapTargetsProperty|TestFMAdversarialProperty|TestFMScalarsRandomProperty|TestFMUnseenRandomProperty)$' ./binding/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
