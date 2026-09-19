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
echo "6b28c156d46f7bb76a37b88c834b53c778928a8b9d4835143d9bd57c086d0365  $HIDDEN/txnkv/transaction/pessimisticlock_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/txnkv/transaction/pessimisticlock_bb_prop_test.go"
install_hidden "txnkv/transaction/pessimisticlock_bb_prop_test.go"
echo "6b28c156d46f7bb76a37b88c834b53c778928a8b9d4835143d9bd57c086d0365  /app/txnkv/transaction/pessimisticlock_bb_prop_test.go" | sha256sum -c --status || checksum_fail "txnkv/transaction/pessimisticlock_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestPessimisticLockDedupProperty|TestPessimisticLockReturnValuesProperty|TestPessimisticLockIfExistsProperty|TestPessimisticLockPrimaryProperty|TestPessimisticLockContractExamples|TestPessimisticLockUnmentionedRandom)$' ./txnkv/transaction/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
