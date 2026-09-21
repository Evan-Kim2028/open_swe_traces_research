#!/usr/bin/env bash
# Shim. The PTY/stdin/model knowledge moved to scripts/ops/agents.py; the launcher is
# scripts/ops/agent_session.py. Kept so existing callers keep working.
#   grok_job.sh <job-name> <brief-path> <workdir> [timeout_sec]
set -u
exec uv run python "$(dirname "$0")/agent_session.py" grok "$1" "$2" "$3" "${4:-14400}"
