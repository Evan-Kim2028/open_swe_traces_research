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
echo "465c3250e0cb08f8d357e31d88263e966368ea4b2c75200d735353077daa6e82  $HIDDEN/render/jsonrenders_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/render/jsonrenders_bb_prop_test.go"
install_hidden "render/jsonrenders_bb_prop_test.go"
echo "465c3250e0cb08f8d357e31d88263e966368ea4b2c75200d735353077daa6e82  /app/render/jsonrenders_bb_prop_test.go" | sha256sum -c --status || checksum_fail "render/jsonrenders_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestJSONWriteJSONContentTypeProperty|TestJSONMarshalErrorProperty|TestIndentedJSONIndentProperty|TestSecureJSONPrefixProperty|TestJsonpJSONCallbackProperty|TestJsonpJSONEmptyCallbackProperty|TestAsciiJSONEscapeProperty|TestPureJSONNoHTMLEscapeProperty|TestJSONPreservesPresetContentTypeProperty)$' ./render/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
