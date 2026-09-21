#!/usr/bin/env bash
# Dispatch one Grok authoring session for a repo, in a SCRATCH dir.
#
# Grok's build CLI hangs in a git worktree - 148 bytes of terminal init, no deliverable,
# reproduced again today. It works normally in a plain directory, so authoring runs on an
# extracted copy of the repo source and the units are harvested back afterwards.
#
#   grok_author.sh <repo> <n_units> [timeout_sec]
set -u
REPO="$1"; N="${2:-10}"; T="${3:-10800}"
R=/home/evan/Documents/open_swe_traces_research
S=/home/evan/.claude/jobs/a1eaeb86/tmp/grokauth_$REPO
IMG="ladder-base:$REPO"
docker image inspect "$IMG" >/dev/null 2>&1 || IMG="ladder-base:${REPO}-obf"
docker image inspect "$IMG" >/dev/null 2>&1 || { echo "no image for $REPO"; exit 1; }

rm -rf "$S"; mkdir -p "$S/outputs" "$S/units"
echo "extracting $IMG -> $S/src"
cid=$(docker create "$IMG") || exit 1
docker cp "$cid:/app" "$S/src" >/dev/null 2>&1
docker rm "$cid" >/dev/null 2>&1
[ -d "$S/src" ] || { echo "extract failed"; exit 1; }
echo "  $(find "$S/src" -name '*.go' -not -name '*_test.go' | wc -l) source files"

# what the bank has already taken, so this session does not re-cut it
( cd "$R" && timeout 300 uv run python scripts/ops/excision_registry.py "$REPO" --md ) \
  > "$S/CLAIMED.md" 2>/dev/null || echo "## Already claimed\n\nNothing yet." > "$S/CLAIMED.md"

cat > "$S/BRIEF.md" <<BRIEF
# Job: author $N synthetic SWE tasks from $REPO

Work in this directory. The repository source is in \`src/\`. Write each task to
\`units/<name>/\`. Do not modify \`src/\` - treat it as read-only reference.

## What a task is

Pick a self-contained behaviour in \`src/\`, cut its implementation out, and write the
artifacts a solver and a grader need. The task must be **hard but fair**: a competent
engineer given the bug report alone should usually fail, and given a full prose contract
should usually succeed.

For each unit \`units/<name>/_author/\`:

- \`gold.patch\` - the diff that RESTORES the excised behaviour (unified diff against src/, \`--- a/<path>\`)
- \`excised/excision.patch\` - the diff that REMOVES it (the inverse)
- \`api.md\` - exported signatures left behind after excision
- \`DETAILS.md\` - numbered list of every behavioural commitment. Annotate each line
  \`Inferable: yes|doc|partially|no\`. For \`no\`, the tests must assert SHAPE, never an
  exact literal - an exact error string or print layout is unguessable and makes the task unfair.
- \`bugreport.md\` - what a user would report. Symptom only. No file names, no line numbers,
  no private identifiers, no fix. This is the L0 prompt.
- \`cheat.patch\` - a minimal special-case that satisfies only the worked examples in the
  bug report. It must be much smaller than gold; if it approaches gold's size you have
  written an implementation, not a cheat.
- \`closure.md\` - one paragraph: what was cut and why a solver cannot simply recall it.

## Choose well - this is most of the value

- **Do not cut famous or memorised code.** An earlier cross-repo round got 17% yield
  instead of 46% because it excised well-known utilities: the solver had already
  memorised them, so the task was trivial. Ask: *could a competent engineer write this
  from the name and signature alone?* If yes, pick something else.
- Prefer behaviour with **arbitrary-but-consistent rules** - encodings, orderings,
  precedence, state transitions - that must be inferred from call sites.
- Prefer code with **3+ distinct call sites** using different arguments.
- Spread out. $N units in $N different files beats $N units in one file.

$(cat "$S/CLAIMED.md")

## Deliverable

\`outputs/grokauth_$REPO.md\`: one row per unit - name, file, what was cut, why it is not
recallable, Inferable breakdown, cheat/gold size ratio. Then stop.
BRIEF

echo "launching grok for $REPO ($N units, timeout ${T}s)"
setsid nohup bash "$R/scripts/ops/grok_job.sh" "grokauth_$REPO" "$S/BRIEF.md" "$S" "$T" \
  > /dev/null 2>&1 &
echo "  scratch: $S"
