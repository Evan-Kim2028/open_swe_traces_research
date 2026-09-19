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
echo "73111e4440f572ed505339c784be9e758d1ccf5e5d96e7912ebdd7cd55fa073c  $HIDDEN/ginS/defaultengine_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/ginS/defaultengine_bb_prop_test.go"
install_hidden "ginS/defaultengine_bb_prop_test.go"
echo "73111e4440f572ed505339c784be9e758d1ccf5e5d96e7912ebdd7cd55fa073c  /app/ginS/defaultengine_bb_prop_test.go" | sha256sum -c --status || checksum_fail "ginS/defaultengine_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestDEContractTableProperty|TestDEHTTPMethodsProperty|TestDESharedRoutesProperty|TestDEGroupMiddlewareProperty|TestDENoRouteNoMethodProperty|TestDEStaticProperty|TestDEHTMLGlobProperty|TestDEAdversarialLazyInitProperty|TestDEUnseenRandomProperty)$' ./ginS/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
