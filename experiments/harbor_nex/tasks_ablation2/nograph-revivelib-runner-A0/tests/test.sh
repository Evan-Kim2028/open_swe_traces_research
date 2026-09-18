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
echo "bd2d5b6afe0a30d632d2229fa2c3cf0cf2068f1e618fb24512d56574d16ac520  $HIDDEN/revivelib/core_test.go" | sha256sum -c --status || checksum_fail "hidden/revivelib/core_test.go"
install_hidden "revivelib/core_test.go"
echo "bd2d5b6afe0a30d632d2229fa2c3cf0cf2068f1e618fb24512d56574d16ac520  /app/revivelib/core_test.go" | sha256sum -c --status || checksum_fail "revivelib/core_test.go"
echo "32dc2d4968ccf9a86a05f4ac9586ea041efe799c1e6f13c3fca52534ef2b668f  $HIDDEN/revivelib/core_internal_test.go" | sha256sum -c --status || checksum_fail "hidden/revivelib/core_internal_test.go"
install_hidden "revivelib/core_internal_test.go"
echo "32dc2d4968ccf9a86a05f4ac9586ea041efe799c1e6f13c3fca52534ef2b668f  /app/revivelib/core_internal_test.go" | sha256sum -c --status || checksum_fail "revivelib/core_internal_test.go"
if go test -count=1 -timeout 15m -run '^(TestReviveLint|TestReviveFormat|TestReviveCreateInstance)$' ./revivelib/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
