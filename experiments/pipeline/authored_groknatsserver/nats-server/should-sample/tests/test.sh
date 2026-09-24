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
echo "f7bb91944a1252d9ba5834091fab9d9bc437f6ebf2d23aa2783a2387bcc0e1d5  $HIDDEN/server/should_sample_bb_test.go" | sha256sum -c --status || checksum_fail "server/should_sample_bb_test.go"
install_hidden "server/should_sample_bb_test.go"
echo "f7bb91944a1252d9ba5834091fab9d9bc437f6ebf2d23aa2783a2387bcc0e1d5  /app/server/should_sample_bb_test.go" | sha256sum -c --status || checksum_fail "server/should_sample_bb_test.go"
if go test -count=1 -timeout 15m -run '^TestDetail' ./server/... ; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
