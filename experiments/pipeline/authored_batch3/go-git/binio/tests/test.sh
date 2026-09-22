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
echo "b6752a396123db20867ff05d99751ad9786098f674a0f738a4766b2cac8409ad  $HIDDEN/utils/binary/binio_bb_test.go" | sha256sum -c --status || checksum_fail "utils/binary/binio_bb_test.go"
install_hidden "utils/binary/binio_bb_test.go"
echo "b6752a396123db20867ff05d99751ad9786098f674a0f738a4766b2cac8409ad  /app/utils/binary/binio_bb_test.go" | sha256sum -c --status || checksum_fail "utils/binary/binio_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./utils/binary/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
