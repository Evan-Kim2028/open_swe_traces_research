#!/usr/bin/env python3
"""Teach harbor's grok-build agent to use this machine's `grok login`.

The agent authenticates only via XAI_API_KEY, a real xAI API key. What this machine has is
a grok.com session token in ~/.grok/auth.json -- which the CLI accepts happily, and which
api.x.ai accepts as a raw bearer, but which the CLI REJECTS when handed to it as
XAI_API_KEY. Verified directly: an isolated HOME plus XAI_API_KEY=<session token> gives
"Not signed in", while the same HOME with auth.json copied in runs grok-4.7 fine.

So the container needs the file, not the variable. That is the whole fix: after the CLI is
installed, write $GROK_AUTH_JSON to ~/.grok/auth.json. The value travels as an environment
variable (--ae), never as a command argument, so it does not appear in process listings --
the same exposure the agent already documents for XAI_API_KEY.

This edits a file under site-packages, so `uv tool upgrade harbor` will silently undo it
and every grok trial will start erroring again at 81% like the 09-20 batch did. Hence a
script rather than a hand edit: re-run it after any harbor upgrade. Idempotent.

    grok_build_patch.py            # apply
    grok_build_patch.py --check    # is it applied?
"""

from __future__ import annotations

import pathlib
import sys

AGENT = pathlib.Path(
    "/home/evan/.local/share/uv/tools/harbor/lib/python3.14/site-packages/"
    "harbor/agents/installed/grok_build.py")

MARKER = "GROK_AUTH_JSON"

ANCHOR = """        skills_command = self._build_register_skills_command()"""

INJECT = '''        # Use this machine's `grok login` instead of requiring an xAI API key. The CLI
        # rejects a grok.com session token passed as XAI_API_KEY but accepts the same
        # token in ~/.grok/auth.json, so hand it the file. Value arrives via the
        # environment (--ae GROK_AUTH_JSON=...), never as an argument, so it stays out of
        # process listings. Applied by scripts/ops/grok_build_patch.py -- re-run that
        # after any `uv tool upgrade harbor`.
        await self.exec_as_agent(
            environment,
            command=(
                'if [ -n "${GROK_AUTH_JSON:-}" ]; then '
                "mkdir -p \\"$HOME/.grok\\" && "
                'printf "%s" "$GROK_AUTH_JSON" > "$HOME/.grok/auth.json" && '
                'chmod 600 "$HOME/.grok/auth.json"; fi'
            ),
        )
        skills_command = self._build_register_skills_command()'''


def main() -> int:
    if not AGENT.exists():
        print(f"agent not found: {AGENT}")
        return 1
    src = AGENT.read_text()
    applied = MARKER in src
    if "--check" in sys.argv:
        print(f"grok_build patch: {'APPLIED' if applied else 'NOT applied'}")
        return 0 if applied else 1
    if applied:
        print("already applied")
        return 0
    if ANCHOR not in src:
        print("anchor not found — harbor's install() has changed; patch by hand")
        return 1
    bak = AGENT.with_suffix(".py.orig")
    if not bak.exists():
        bak.write_text(src)
    AGENT.write_text(src.replace(ANCHOR, INJECT, 1))
    print(f"patched {AGENT}\n  backup at {bak}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
