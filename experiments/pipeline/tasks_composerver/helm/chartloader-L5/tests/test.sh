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
echo "049ef52320e2dd3838c8cd700de8afaac8acc3eeabdf2ce34b7ec46e13ddc5b3  $HIDDEN/internal/chart/v3/loader/chartloader_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/chart/v3/loader/chartloader_bb_prop_test.go"
install_hidden "internal/chart/v3/loader/chartloader_bb_prop_test.go"
echo "049ef52320e2dd3838c8cd700de8afaac8acc3eeabdf2ce34b7ec46e13ddc5b3  /app/internal/chart/v3/loader/chartloader_bb_prop_test.go" | sha256sum -c --status || checksum_fail "internal/chart/v3/loader/chartloader_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestCLContractTable|TestCLMergeMapsProperty|TestCLLoadValuesProperty|TestCLLoadFilesProperty|TestCLLoaderEquivalence|TestCLBOMProperty|TestCLAdversarialPaths)$' ./internal/chart/v3/loader/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
