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
echo "b17ba35de94cd0ad31ffdf0747a7c3546d0844cdaa3b430f700ea87a37b7bf43  $HIDDEN/render/htmlrender_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/render/htmlrender_bb_prop_test.go"
install_hidden "render/htmlrender_bb_prop_test.go"
echo "b17ba35de94cd0ad31ffdf0747a7c3546d0844cdaa3b430f700ea87a37b7bf43  /app/render/htmlrender_bb_prop_test.go" | sha256sum -c --status || checksum_fail "render/htmlrender_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestHTMLProductionNamedTemplateProperty|TestHTMLProductionEmptyNameProperty|TestHTMLNotConfiguredProperty|TestHTMLDebugFilesProperty|TestHTMLDebugGlobProperty|TestHTMLDebugFSProperty|TestHTMLDebugPanicProperty|TestHTMLContentTypeProperty|TestHTMLDelimsProperty|TestHTMLExecuteErrorProperty)$' ./render/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
