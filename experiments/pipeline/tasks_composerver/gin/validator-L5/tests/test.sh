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
echo "6bb695226c59880bfd61838baf84226cfa91e7bd9dcd70634cfed4d37fcd6bee  $HIDDEN/binding/validator_bb_prop_test.go" | sha256sum -c --status || checksum_fail "hidden/binding/validator_bb_prop_test.go"
install_hidden "binding/validator_bb_prop_test.go"
echo "6bb695226c59880bfd61838baf84226cfa91e7bd9dcd70634cfed4d37fcd6bee  /app/binding/validator_bb_prop_test.go" | sha256sum -c --status || checksum_fail "binding/validator_bb_prop_test.go"
if go test -count=1 -timeout 15m -run '^(TestBBValidateNil|TestBBValidateNilRandom|TestBBValidatePrimitives|TestBBValidatePrimitivesRandom|TestBBValidatePointerToNonStruct|TestBBValidateStructRules|TestBBValidateStructRandom|TestBBValidateSlicePass|TestBBValidateSliceRandomPass|TestBBSliceValidationErrorFormat|TestBBValidateSliceFail|TestBBValidatorEngineCustom|TestBBValidatorEngineLazyInit)$' ./binding/...; then
  :
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
echo 1 > /logs/verifier/reward.txt
exit 0
