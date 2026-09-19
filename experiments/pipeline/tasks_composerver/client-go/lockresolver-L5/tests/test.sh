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
echo "4af21e4d9825d5db91f2018cc30e856cef8c7072d4552e825f45db5f653ed362  $HIDDEN/txnkv/txnlock/lockresolver_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/txnkv/txnlock/lockresolver_bb_prop_test.go"
install_hidden "txnkv/txnlock/lockresolver_bb_prop_test.go"
echo "4af21e4d9825d5db91f2018cc30e856cef8c7072d4552e825f45db5f653ed362  /app/txnkv/txnlock/lockresolver_bb_prop_test.go" | sha256sum -c --status || checksum_fail "txnkv/txnlock/lockresolver_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestExtractLockFromKeyErrProperty|TestResolvingLocksTrackingProperty|TestLockResolverTxnStatusProperty|TestLockResolverReadPathProperty|TestLockResolverContractExamples|TestLockResolverUnmentionedRandom)$' ./txnkv/txnlock/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
