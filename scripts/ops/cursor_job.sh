#!/usr/bin/env bash
# Shim. Key handling and the bare-slug requirement moved to scripts/ops/agents.py; the
# launcher is scripts/ops/agent_session.py.
#   cursor_job.sh <job-name> <brief-path> <workdir> [timeout_sec]
set -u
exec uv run python "$(dirname "$0")/agent_session.py" cursor "$1" "$2" "$3" "${4:-10800}"
