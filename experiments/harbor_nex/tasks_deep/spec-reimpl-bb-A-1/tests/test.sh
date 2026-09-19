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
echo "2102e575748f37d20c1ae81d1a49bc7c916b4de8e7da35e6349ca225774245bf  $HIDDEN/internal/apicodec/codec_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/apicodec/codec_bb_prop_test.go"
install_hidden "internal/apicodec/codec_bb_prop_test.go"
echo "2102e575748f37d20c1ae81d1a49bc7c916b4de8e7da35e6349ca225774245bf  /app/internal/apicodec/codec_bb_prop_test.go" | sha256sum -c --status || checksum_fail "internal/apicodec/codec_bb_prop_test.go"
echo "3a824afa3541048c0c90c22ee70fc8bed8914c6223ce86225088b3373e78a739  $HIDDEN/internal/apicodec/codec_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/apicodec/codec_test.go"
install_hidden "internal/apicodec/codec_test.go"
echo "3a824afa3541048c0c90c22ee70fc8bed8914c6223ce86225088b3373e78a739  /app/internal/apicodec/codec_test.go" | sha256sum -c --status || checksum_fail "internal/apicodec/codec_test.go"
if go test -count=1 -timeout 15m -run '^(TestCodecKeyRangeRoundTrip|TestCodecClipProperties|TestCodecContractExamples|TestCodecUnmentionedRandom|TestParseKeyspaceID|TestDecodeKey|TestEncodeUnknownRequest)$' ./internal/apicodec/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
