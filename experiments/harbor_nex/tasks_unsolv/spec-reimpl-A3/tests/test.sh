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
echo "7c380e9ea62815028bf84ccd09a35960f6b5abf514e4fc01aaff89923455b684  $HIDDEN/internal/apicodec/codec_v2_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/apicodec/codec_v2_test.go"
install_hidden "internal/apicodec/codec_v2_test.go"
echo "7c380e9ea62815028bf84ccd09a35960f6b5abf514e4fc01aaff89923455b684  /app/internal/apicodec/codec_v2_test.go" | sha256sum -c --status || checksum_fail "internal/apicodec/codec_v2_test.go"
echo "3a824afa3541048c0c90c22ee70fc8bed8914c6223ce86225088b3373e78a739  $HIDDEN/internal/apicodec/codec_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/apicodec/codec_test.go"
install_hidden "internal/apicodec/codec_test.go"
echo "3a824afa3541048c0c90c22ee70fc8bed8914c6223ce86225088b3373e78a739  /app/internal/apicodec/codec_test.go" | sha256sum -c --status || checksum_fail "internal/apicodec/codec_test.go"
echo "6b4ce1af1ae4de720cc94e8defcc4c08967d5903758effe0047a2bcbb6a4f2e1  $HIDDEN/internal/apicodec/decode_fatal_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/apicodec/decode_fatal_test.go"
install_hidden "internal/apicodec/decode_fatal_test.go"
echo "6b4ce1af1ae4de720cc94e8defcc4c08967d5903758effe0047a2bcbb6a4f2e1  /app/internal/apicodec/decode_fatal_test.go" | sha256sum -c --status || checksum_fail "internal/apicodec/decode_fatal_test.go"
if go test -count=1 -timeout 15m -run '^(TestCodecV2|TestEncodeRequest|TestEncodeV2KeyRanges|TestNewCodecV2|TestDecodeEpochNotMatch|TestGetKeyspaceID|TestEncodeMPPRequest|TestDecodeBucketKeys|TestParseKeyspaceID|TestDecodeKey|TestEncodeUnknownRequest|TestMalformedRegionKeyIsDecodeError)$' ./internal/apicodec/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
