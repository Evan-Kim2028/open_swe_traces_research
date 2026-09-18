#!/bin/bash
set -uo pipefail
mkdir -p /logs/verifier
cd /app
if go test -count=1 -timeout 15m -run '^(TestDecodeKey)$' ./internal/apicodec/...; then
  echo 1 > /logs/verifier/reward.txt
  exit 0
else
  echo 0 > /logs/verifier/reward.txt
  exit 1
fi
