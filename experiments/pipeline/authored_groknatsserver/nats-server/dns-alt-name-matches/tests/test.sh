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
echo "33847ba8e9ffb9c8c993af8b38e2920df1acca8e7e790964518414358c60c904  $HIDDEN/server/dns_alt_name_matches_bb_test.go" | sha256sum -c --status || checksum_fail "server/dns_alt_name_matches_bb_test.go"
install_hidden "server/dns_alt_name_matches_bb_test.go"
echo "33847ba8e9ffb9c8c993af8b38e2920df1acca8e7e790964518414358c60c904  /app/server/dns_alt_name_matches_bb_test.go" | sha256sum -c --status || checksum_fail "server/dns_alt_name_matches_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./server/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
