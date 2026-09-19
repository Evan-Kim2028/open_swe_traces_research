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
echo "4297e404e091173f3933d5a9a76d2801db46c3f0ab8b9514b1afc7e6ab5b2b1d  $HIDDEN/wirerpc/interceptor/chain_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/wirerpc/interceptor/chain_bb_prop_test.go"
install_hidden "wirerpc/interceptor/chain_bb_prop_test.go"
echo "4297e404e091173f3933d5a9a76d2801db46c3f0ab8b9514b1afc7e6ab5b2b1d  /app/wirerpc/interceptor/chain_bb_prop_test.go" | sha256sum -c --status || checksum_fail "wirerpc/interceptor/chain_bb_prop_test.go"
echo "ff5f57d2f43d5d1d25252b4a0cdcf59d0d9fa237e3dd8acb4bbfcde7846d1f2a  $HIDDEN/internal/client/client_interceptor_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/client/client_interceptor_test.go"
install_hidden "internal/client/client_interceptor_test.go"
echo "ff5f57d2f43d5d1d25252b4a0cdcf59d0d9fa237e3dd8acb4bbfcde7846d1f2a  /app/internal/client/client_interceptor_test.go" | sha256sum -c --status || checksum_fail "internal/client/client_interceptor_test.go"
if go test -count=1 -timeout 15m -run '^(TestInterceptorChainOrder|TestInterceptorChainDedupAndCompose|TestInterceptorContractExamples|TestInterceptorUnmentionedRandom|TestInterceptedClient|TestAppendChainedInterceptor)$' ./wirerpc/interceptor/... ./internal/client/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
