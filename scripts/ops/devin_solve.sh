#!/usr/bin/env bash
# Run Devin as a SOLVER on task dirs, host-side, and score with the unit's own verifier.
#
# Harbor's devin agent runs the CLI INSIDE the trial container, where
# ~/.config/devin/config.json does not exist; the CLI ignores DEVIN_API_KEY and reports an
# empty model list ("Unknown model: 'swe-2-max'"). Running the CLI on the host sidesteps
# that: the agent edits a copy of environment/src, then the patched tree is baked into the
# unit's own image and scored by the unit's own tests/test.sh -- the same verifier harbor
# would use.
#
# Usage: devin_solve.sh <sweep dir> <out.jsonl> [model]
set -u
SWEEP="${1:?sweep dir}"; OUT="${2:?out jsonl}"; MODEL="${3:-swe-2-max}"
export PATH="$HOME/.local/bin:$PATH"
WORK=$(mktemp -d); trap 'rm -rf "$WORK"' EXIT
for u in "$SWEEP"/*/; do
  [ -d "$u/environment/src" ] || continue
  name=$(basename "$u")
  grep -q "\"unit\": \"$name\"" "$OUT" 2>/dev/null && continue
  rm -rf "$WORK/src"; cp -a "$u/environment/src" "$WORK/src"
  start=$(date +%s)
  ( cd "$WORK/src" && timeout 3600 devin --model "$MODEL" --permission-mode yolo --print \
      --respect-workspace-trust false -- "$(cat "$OLDPWD/$u/instruction.md" 2>/dev/null || cat "$u/instruction.md")" \
  ) > "$WORK/agent.log" 2>&1
  rc=$?
  elapsed=$(( $(date +%s) - start ))
  # score with the unit's own verifier
  cp "$u/environment/Dockerfile" "$WORK/Dockerfile"
  img="devinsolve-$(echo "$name" | tr 'A-Z' 'a-z' | tr -cd 'a-z0-9-')"
  reward=null; note=""
  if docker build -q -t "$img" "$WORK" > "$WORK/build.log" 2>&1; then
    rm -rf "$WORK/logs"; mkdir -p "$WORK/logs"
    docker run --rm --network none -v "$(readlink -f "$u/tests")":/tests:ro \
      -v "$WORK/logs":/logs "$img" bash /tests/test.sh > "$WORK/test.log" 2>&1
    reward=$(cat "$WORK/logs/verifier/reward.txt" 2>/dev/null || echo null)
    docker rmi -f "$img" >/dev/null 2>&1
  else
    note="image build failed"
  fi
  printf '{"unit":"%s","reward":%s,"agent_rc":%d,"seconds":%d,"note":"%s","agent_tail":%s}\n' \
    "$name" "${reward:-null}" "$rc" "$elapsed" "$note" \
    "$(tail -c 300 "$WORK/agent.log" | python3 -c 'import json,sys;print(json.dumps(sys.stdin.read()))')" >> "$OUT"
  echo "$name reward=$reward rc=$rc ${elapsed}s $note"
done
