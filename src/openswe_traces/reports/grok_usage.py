"""Grok usage outside the trial ledger: grok-build sessions run on the host.

    uv run python -m openswe_traces.reports.grok_usage

The ledger prices Grok's harbor trials from result.json. Grok also authored units in the
oswt-* worktrees and ran analysis from the research repo, and those sessions live only in
~/.grok/sessions/<project>/<session>/usage.json. grok-build writes exact counts there:
input (cached reads included, as Composer counts them), output, and cost in ticks of
1e-10 USD.

A session belongs to Stage 1 when its project directory is this repository, an oswt-*
worktree, or a grokauth_* scratch directory. Sessions run from ~/Documents cover several
projects at once, so the ones that worked on this one are listed by id below, attributed
by what their transcripts are about.
"""
import collections
import json
import pathlib
import re
import urllib.parse

SESSIONS = pathlib.Path.home() / ".grok" / "sessions"
TICKS_PER_USD = 1e10

PROJECT = re.compile(r"open_swe_traces_research|/oswt-|/grokauth_|/grok(?:test\d*|scratch)$")

# ~/Documents sessions whose transcripts are about this repository (research-repo mentions
# far outnumber other projects'). The paper-writing session is kept apart: it wrote the
# write-up, it did not build or grade tasks.
DOCUMENTS_STAGE1 = ("01a0c091", "01a0c098", "01a0c09a", "01a0c48e", "01a0c666")
DOCUMENTS_WRITING = ("01a0c97a",)


def sessions():
    """Yield (project dir, session id, session usage) for every grok-build session."""
    for f in SESSIONS.glob("*/*/usage.json"):
        try:
            usage = json.loads(f.read_text())["session"]
        except (OSError, ValueError, KeyError):
            continue
        yield urllib.parse.unquote(f.parent.parent.name), f.parent.name, usage


def classify(project, sid):
    if PROJECT.search(project):
        return "authoring" if "/oswt-" in project or "/grokauth_" in project else "operations"
    if project.endswith("/Documents"):
        if sid.startswith(DOCUMENTS_STAGE1):
            return "operations"
        if sid.startswith(DOCUMENTS_WRITING):
            return "write-up"
    return None


def report():
    out = collections.defaultdict(collections.Counter)
    for project, sid, u in sessions():
        kind = classify(project, sid)
        if kind is None:
            continue
        c = out[kind]
        c["sessions"] += 1
        c["input"] += u.get("inputTokens") or 0
        c["cache_read"] += u.get("cachedReadTokens") or 0
        c["output"] += u.get("outputTokens") or 0
        c["usd"] += (u.get("costUsdTicks") or 0) / TICKS_PER_USD
    return dict(out)


def cli():
    for kind, c in sorted(report().items()):
        print(f"{kind:11s} sessions={c['sessions']:3d} tokens={(c['input'] + c['output']) / 1e6:7.1f}M "
              f"(cache {c['cache_read'] / 1e6:6.1f}M) ${c['usd']:7.2f}")


if __name__ == "__main__":
    cli()
