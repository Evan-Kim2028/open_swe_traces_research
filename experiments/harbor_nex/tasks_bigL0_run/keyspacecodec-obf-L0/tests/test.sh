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
echo "6a515538980f4e14d39e895ffd0652f4ea03e271c38be3b98a551a3530d122e1  $HIDDEN/internal/apicodec/codec_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/apicodec/codec_bb_prop_test.go"
install_hidden "internal/apicodec/codec_bb_prop_test.go"
echo "6a515538980f4e14d39e895ffd0652f4ea03e271c38be3b98a551a3530d122e1  /app/internal/apicodec/codec_bb_prop_test.go" | sha256sum -c --status || checksum_fail "internal/apicodec/codec_bb_prop_test.go"
echo "a4cbccff7584c573d8b3bd488a7b12a65e46814cb72b5f83c4f3e0adba99554b  $HIDDEN/internal/locate/decode_fatal_bb_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/locate/decode_fatal_bb_test.go"
install_hidden "internal/locate/decode_fatal_bb_test.go"
echo "a4cbccff7584c573d8b3bd488a7b12a65e46814cb72b5f83c4f3e0adba99554b  /app/internal/locate/decode_fatal_bb_test.go" | sha256sum -c --status || checksum_fail "internal/locate/decode_fatal_bb_test.go"
if go test -count=1 -timeout 15m -run '^(TestCodecKeyRangeRoundTrip|TestCodecClipProperties|TestCodecContractExamples|TestCodecUnmentionedRandom|TestLocateMalformedRegionNoBackoff)$' ./internal/apicodec/... ./internal/locate/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
