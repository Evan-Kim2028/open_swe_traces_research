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
echo "386b691d60fefe2a57dff702fb717c18815bf0615d478627a276a74c88036ec2  $HIDDEN/plumbing/format/objfile/objfile_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/objfile/objfile_bb_test.go"
install_hidden "plumbing/format/objfile/objfile_bb_test.go"
echo "386b691d60fefe2a57dff702fb717c18815bf0615d478627a276a74c88036ec2  /app/plumbing/format/objfile/objfile_bb_test.go" | sha256sum -c --status || checksum_fail "plumbing/format/objfile/objfile_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./plumbing/format/objfile/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
