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
echo "74e66138680e3c809992594f622ce9bbc03b3354e319658c1e3d3cb34a207993  $HIDDEN/lint/filefilter_test.go" | sha256sum -c --status || checksum_fail "hidden/lint/filefilter_test.go"
install_hidden "lint/filefilter_test.go"
echo "74e66138680e3c809992594f622ce9bbc03b3354e319658c1e3d3cb34a207993  /app/lint/filefilter_test.go" | sha256sum -c --status || checksum_fail "lint/filefilter_test.go"
echo "5f838dbf69cb068fc7fed8fc96e980dd501b0f45bb87bf473f86aa94d87a6334  $HIDDEN/test/file_filter_test.go" | sha256sum -c --status || checksum_fail "hidden/test/file_filter_test.go"
install_hidden "test/file_filter_test.go"
echo "5f838dbf69cb068fc7fed8fc96e980dd501b0f45bb87bf473f86aa94d87a6334  /app/test/file_filter_test.go" | sha256sum -c --status || checksum_fail "test/file_filter_test.go"
echo "450313a2ac048f9187d183a42f1a10a804217e1c202fccb9c6240918070c463a  $HIDDEN/config/getconfig_bb_test.go" | sha256sum -c --status || checksum_fail "hidden/config/getconfig_bb_test.go"
install_hidden "config/getconfig_bb_test.go"
echo "450313a2ac048f9187d183a42f1a10a804217e1c202fccb9c6240918070c463a  /app/config/getconfig_bb_test.go" | sha256sum -c --status || checksum_fail "config/getconfig_bb_test.go"
if go test -count=1 -timeout 15m -run '^(TestFileFilter|TestFileExcludeFilterAtRuleLevel|TestGetConfig)$' ./lint/... ./test/... ./config/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
