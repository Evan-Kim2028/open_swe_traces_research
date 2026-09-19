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
echo "0082a996ccea65e42cc3c61bb8f9998bf3a6f32ccbe10ef89d26fda34adad802  $HIDDEN/binding/multipartfiles_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/binding/multipartfiles_bb_prop_test.go"
install_hidden "binding/multipartfiles_bb_prop_test.go"
echo "0082a996ccea65e42cc3c61bb8f9998bf3a6f32ccbe10ef89d26fda34adad802  /app/binding/multipartfiles_bb_prop_test.go" | sha256sum -c --status || checksum_fail "binding/multipartfiles_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestBBMultipartPointerFirstFile|TestBBMultipartPointerRandom|TestBBMultipartValueFirstFile|TestBBMultipartSliceCount|TestBBMultipartSliceRandom|TestBBMultipartArrayExactLen|TestBBMultipartArrayLenInvalid|TestBBMultipartWrongTypeError|TestBBMultipartWrongTypeRandom|TestBBMultipartFormValueFallback)$' ./binding/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
