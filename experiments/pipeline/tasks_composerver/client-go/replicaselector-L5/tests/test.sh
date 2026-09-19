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
echo "b0ebb50c9a2dc76232ba40002e01b6e9e3242622f9a0c555e359db118df6e055  $HIDDEN/internal/locate/replicaselector_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/internal/locate/replicaselector_bb_prop_test.go"
install_hidden "internal/locate/replicaselector_bb_prop_test.go"
echo "b0ebb50c9a2dc76232ba40002e01b6e9e3242622f9a0c555e359db118df6e055  /app/internal/locate/replicaselector_bb_prop_test.go" | sha256sum -c --status || checksum_fail "internal/locate/replicaselector_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestReplicaSelectorAccessProperty|TestReplicaSelectorFailoverProperty|TestReplicaSelectorStaleFailoverProperty|TestReplicaSelectorContractExamples)$' ./internal/locate/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
