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
echo "081bbcc7256d76d8a58da28f0b8d4a42aa78517ed4e2bba097ee71c2d9620090  $HIDDEN/server/ws_pmc_extension_bb_test.go" | sha256sum -c --status || checksum_fail "server/ws_pmc_extension_bb_test.go"
install_hidden "server/ws_pmc_extension_bb_test.go"
echo "081bbcc7256d76d8a58da28f0b8d4a42aa78517ed4e2bba097ee71c2d9620090  /app/server/ws_pmc_extension_bb_test.go" | sha256sum -c --status || checksum_fail "server/ws_pmc_extension_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./server/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
