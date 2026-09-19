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
echo "947e15c88ec54c53b96b541ad8795a9a00ad6d275c6150f7ab4d2b2bae14e543  $HIDDEN/pkg/strvals/strvalsparser_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/pkg/strvals/strvalsparser_bb_prop_test.go"
install_hidden "pkg/strvals/strvalsparser_bb_prop_test.go"
echo "947e15c88ec54c53b96b541ad8795a9a00ad6d275c6150f7ab4d2b2bae14e543  /app/pkg/strvals/strvalsparser_bb_prop_test.go" | sha256sum -c --status || checksum_fail "pkg/strvals/strvalsparser_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestStrvalsContractTableProperty|TestStrvalsNoPanicProperty|TestStrvalsParseSimpleProperty|TestStrvalsListGrowProperty|TestStrvalsStringModeProperty|TestStrvalsLiteralProperty|TestStrvalsParseIntoProperty|TestStrvalsLiteralIntoProperty|TestStrvalsJSONProperty|TestStrvalsFileModeProperty|TestStrvalsToYAMLProperty|TestStrvalsNestedLevelProperty|TestStrvalsUnseenRandomProperty)$' ./pkg/strvals/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
