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
echo "f98f7fd74e3ff12b8879336e744b6b4b5abb0540af53d303ece568f989b2316a  $HIDDEN/config/modconfig_bb_test.go" | sha256sum -c --status || checksum_fail "config/modconfig_bb_test.go"
install_hidden "config/modconfig_bb_test.go"
echo "f98f7fd74e3ff12b8879336e744b6b4b5abb0540af53d303ece568f989b2316a  /app/config/modconfig_bb_test.go" | sha256sum -c --status || checksum_fail "config/modconfig_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./config/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
