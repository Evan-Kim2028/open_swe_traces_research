#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
mkdir -p /logs/artifacts
if [ -d /pristine ]; then
  ( cd / && if command -v git >/dev/null 2>&1; then git diff --no-index --no-color pristine app; else diff -ruN pristine app; fi )     | sed -e 's|a/pristine/|a/|g' -e 's|b/app/|b/|g' -e 's|a/app/|a/|g' -e 's|b/pristine/|b/|g' -e 's|^--- pristine/|--- a/|' -e 's|^+++ app/|+++ b/|'     > /logs/artifacts/agent.patch || true
fi
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
echo "b8034778f71d821eea29f1d0f05e22602d28d1b1a6173216c0416bfd9e4f897f  $HIDDEN/upup/pkg/fi/cloudup/openstack/openstackmetadata/osmetadata_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/upup/pkg/fi/cloudup/openstack/openstackmetadata/osmetadata_bb_prop_test.go"
install_hidden "upup/pkg/fi/cloudup/openstack/openstackmetadata/osmetadata_bb_prop_test.go"
echo "b8034778f71d821eea29f1d0f05e22602d28d1b1a6173216c0416bfd9e4f897f  /app/upup/pkg/fi/cloudup/openstack/openstackmetadata/osmetadata_bb_prop_test.go" | sha256sum -c --status || checksum_fail "upup/pkg/fi/cloudup/openstack/openstackmetadata/osmetadata_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestOsmetadataDefaultSearchOrderProperty|TestOsmetadataGetLocalProperty|TestOsmetadataJSONRoundTripProperty|TestOsmetadataJSONRandomProperty)$' ./upup/pkg/fi/cloudup/openstack/openstackmetadata/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
