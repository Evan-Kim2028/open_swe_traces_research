#!/usr/bin/env bash
# verifyunit.sh <unit>
# In a scratch copy of the pristine tree:
#   excision -> build + full pkg test green; +gold -> build + closure tests pass;
#   +cheat -> build + closure tests FAIL. Closure tests = testre in spec, run
#   against the ORIGINAL test files restored from index.
set -uo pipefail
TOOLS="$(cd "$(dirname "$0")" && pwd)"
SRC="$TOOLS/../../work/go-github/scratch"
ROOT="$TOOLS/../go-github"
unit=$1
U="$ROOT/$unit/_author"
SCR="/tmp/verify3/$unit"
TESTRE=$(python3 -c "import json,sys; print(json.load(open('$TOOLS/spec/$unit.json')).get('testre','.'))")
rm -rf "$SCR" && mkdir -p "$SCR"
cp -a "$SRC/." "$SCR/"
cd "$SCR"
export GOPROXY=off

echo "=== $unit: excision ==="
git apply --whitespace=nowarn "$U/excised/excision.patch" && echo APPLY_EXC_OK || { echo APPLY_EXC_FAIL; exit 1; }
go build ./github >/dev/null 2>&1 && echo BUILD_EXC_OK || { echo BUILD_EXC_FAIL; go build ./github 2>&1 | head -10; }
go vet ./github >/dev/null 2>&1 && echo VET_EXC_OK || echo VET_EXC_FAIL
go test -count=1 ./github 2>&1 | tail -5

echo "=== $unit: gold ==="
git apply --whitespace=nowarn "$U/gold.patch" && echo APPLY_GOLD_OK || { echo APPLY_GOLD_FAIL; exit 1; }
go build ./github >/dev/null 2>&1 && echo BUILD_GOLD_OK || { echo BUILD_GOLD_FAIL; go build ./github 2>&1 | head -10; }
# restore ORIGINAL test files (brings back the deleted closure tests)
python3 - "$TOOLS" "$unit" <<'EOF'
import json, subprocess, sys
tools, unit = sys.argv[1], sys.argv[2]
spec = json.load(open(f"{tools}/spec/{unit}.json"))
for rel in spec.get("deltests", {}):
    out = subprocess.run(["git","show",f":{rel}"],capture_output=True,text=True).stdout
    open(rel,"w").write(out)
EOF
go test -count=1 -run "$TESTRE" ./github 2>&1 | tail -6

echo "=== $unit: cheat (expect test FAIL) ==="
git apply -R --whitespace=nowarn "$U/gold.patch"
git apply --whitespace=nowarn "$U/cheat.patch" && echo APPLY_CHEAT_OK || { echo APPLY_CHEAT_FAIL; exit 1; }
go build ./github >/dev/null 2>&1 && echo BUILD_CHEAT_OK || { echo BUILD_CHEAT_FAIL; go build ./github 2>&1 | head -10; }
go test -count=1 -run "$TESTRE" ./github 2>&1 | tail -6

echo "=== sizes ==="
c=$(grep -c '^+[^+]' "$U/cheat.patch" || true)
g=$(grep -c '^+[^+]' "$U/gold.patch" || true)
python3 -c "print(f'$unit cheat=$c gold=$g ratio={$c/$g:.2f}' if $g else '$unit cheat=$c gold=$g')"
