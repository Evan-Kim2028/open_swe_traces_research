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
echo "e1cf8837733a4f33de4bc99e2e2185d25440d86cbbd7721e33aa250aa34d6046  $HIDDEN/internal/resolver/depresolver_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/resolver/depresolver_bb_prop_test.go"
install_hidden "internal/resolver/depresolver_bb_prop_test.go"
echo "e1cf8837733a4f33de4bc99e2e2185d25440d86cbbd7721e33aa250aa34d6046  /app/internal/resolver/depresolver_bb_prop_test.go" | sha256sum -c --status || checksum_fail "internal/resolver/depresolver_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestDRContractTable|TestDRResolveScenarios|TestDRHashReqStable|TestDRHashV2Stable|TestDRGetLocalPathProperty)$' ./internal/resolver/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
