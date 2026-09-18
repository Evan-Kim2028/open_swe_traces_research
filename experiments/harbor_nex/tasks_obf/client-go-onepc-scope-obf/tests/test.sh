#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
fail() {
  echo 0 > /logs/verifier/reward.txt
  exit 1
}
cd /app/integration_tests
expected='ae74411e252cda9c97e8fd7b49573132a8db493d7b0ed8e262efa1dd078d5ed3'
actual=$(find . -name '*_test.go' | LC_ALL=C sort | xargs -r sha256sum | sha256sum | awk '{print $1}')
if [ "$actual" != "$expected" ]; then
  echo "consumer test files were modified; the library must be fixed" >&2
  fail
fi
if go test -ldflags=-checklinkname=0 -count=1 -timeout 15m -run '^(TestOnePC)$' .; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  fail
fi
