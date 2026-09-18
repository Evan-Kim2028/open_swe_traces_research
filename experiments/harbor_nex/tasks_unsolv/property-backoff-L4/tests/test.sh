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
echo "f5cc993b7b573c889267a2910ab0274997422f181f9973fa8f4d74b6472aea6d  $HIDDEN/internal/client/retry/backoff_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/client/retry/backoff_prop_test.go"
install_hidden "internal/client/retry/backoff_prop_test.go"
echo "f5cc993b7b573c889267a2910ab0274997422f181f9973fa8f4d74b6472aea6d  /app/internal/client/retry/backoff_prop_test.go" | sha256sum -c --status || checksum_fail "internal/client/retry/backoff_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestBackoffExponentialProperty)$' ./internal/client/retry/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
