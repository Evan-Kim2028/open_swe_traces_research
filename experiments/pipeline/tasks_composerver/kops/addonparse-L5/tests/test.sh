#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
mkdir -p /logs/artifacts
if [ -d /pristine ]; then
  ( cd / && if command -v git >/dev/null 2>&1; then git diff --no-index --no-color pristine app; else diff -ruN pristine app; fi )     | sed -e 's|a/pristine/|a/|g' -e 's|b/app/|b/|g' -e 's|a/app/|a/|g' -e 's|b/pristine/|b/|g' -e 's|^--- pristine/|--- a/|' -e 's|^+++ app/|+++ b/|'     > /logs/artifacts/agent.patch || true
fi
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
echo "67302f0f8c15eab3880414a7305c78d99ea2649fdaae119dfc3bde43d27b17a9  $HIDDEN/channels/pkg/channels/addonparse_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/channels/pkg/channels/addonparse_bb_prop_test.go"
install_hidden "channels/pkg/channels/addonparse_bb_prop_test.go"
echo "67302f0f8c15eab3880414a7305c78d99ea2649fdaae119dfc3bde43d27b17a9  /app/channels/pkg/channels/addonparse_bb_prop_test.go" | sha256sum -c --status || checksum_fail "channels/pkg/channels/addonparse_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestAddonParseContractTableProperty|TestAddonParseEmptyInvalidProperty|TestAddonGetCurrentVersionProperty|TestAddonReplacementProperty|TestAddonRequiredUpdatesProperty)$' ./channels/pkg/channels/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
