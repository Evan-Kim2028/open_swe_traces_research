#!/usr/bin/env bash
# Launch one Grok build-CLI job headlessly.
#
# The CLI is a TUI. Three things are required, each learned by it hanging:
#   1. a PTY            - without one it dies with "No such device or address (os error 6)"
#   2. headless output  - or it renders into an alternate screen buffer and logs nothing
#   3. a stdin that never reaches EOF, owned by THIS process group. An earlier version used
#      process substitution < <(tail -f /dev/null); that pipe belongs to the launching
#      shell, so when the launcher returned the pipe tore down and five jobs sat dead for
#      five minutes with 148 bytes of terminal-init in their logs. A piped `sleep infinity`
#      keeps the writer inside the job's own group, alive as long as the job.
#
#   grok_job.sh <job-name> <brief-path> <worktree> [timeout_sec]
set -u
J="$1"; B="$2"; W="$3"; T="${4:-14400}"
mkdir -p "$W/outputs"
cd "$W" || exit 1
PROMPT="Read the file $B and carry out the job it describes, in full, in this working directory. Follow every rule in it. When the deliverable is written, stop."
exec timeout "$T" bash -c \
  "sleep infinity | script -qec \"grok --model grok-4.6 --always-approve --output-format streaming-json '\$0'\" /dev/null" \
  "$PROMPT" > "$W/outputs/$J.grok.log" 2>&1
