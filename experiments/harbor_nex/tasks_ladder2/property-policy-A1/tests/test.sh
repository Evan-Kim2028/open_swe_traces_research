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
echo "a8ffa3e20a96507fa70e85f1fbb318598a2b7190bb163707f2d935d604393e8f  $HIDDEN/internal/client/retry/policy_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/client/retry/policy_prop_test.go"
install_hidden "internal/client/retry/policy_prop_test.go"
echo "a8ffa3e20a96507fa70e85f1fbb318598a2b7190bb163707f2d935d604393e8f  /app/internal/client/retry/policy_prop_test.go" | sha256sum -c --status || checksum_fail "internal/client/retry/policy_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestBackoffPolicyTableProperty|TestBackoffPolicyContractExamples|TestBackoffPolicyUnmentionedRandom)$' ./internal/client/retry/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
