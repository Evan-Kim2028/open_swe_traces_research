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
echo "7590b11c203b1a17e0787b2e7805e203e91442065eec8c1d373261b44a1e6479  $HIDDEN/pkg/repo/v1/repindex_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/pkg/repo/v1/repindex_bb_prop_test.go"
install_hidden "pkg/repo/v1/repindex_bb_prop_test.go"
echo "7590b11c203b1a17e0787b2e7805e203e91442065eec8c1d373261b44a1e6479  /app/pkg/repo/v1/repindex_bb_prop_test.go" | sha256sum -c --status || checksum_fail "pkg/repo/v1/repindex_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestRIContractTable|TestRIAddSort|TestRIGet|TestRIMerge|TestRIIndexDirectory|TestRILoadRoundTrip|TestRILoadMalformed)$' ./pkg/repo/v1/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
