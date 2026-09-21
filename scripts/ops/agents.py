#!/usr/bin/env python3
"""One registry for every headless agent connector.

Each agent's quirks were discovered painfully and then written down in exactly one
script, where nothing else could find them. That cost real money tonight:

  * Devin needs `server.codeium.com` reachable or its in-container model list comes back
    EMPTY and every trial dies with "Unknown model: swe-2-max" - which reads as a bad
    slug, not a blocked host. The requirement lived in sweep_seq.sh; the egress allowlist
    lived in affordance.py; nothing joined them, and 98 trials died in one hour.
  * cursor-agent must NOT inherit a stale CURSOR_API_KEY (a free-plan key kills every run
    with "Named models unavailable"), and takes the BARE slug composer-2.5 - "cursor/
    composer-2.5" is harbor's naming and is rejected. That knowledge lived in sweep_seq.sh
    only, so 26 direct harbor trials died.
  * grok needs a PTY, headless output, and a stdin that never EOFs owned by its own
    process group - and hangs in a git worktree, working only in a plain directory.

Import this instead of re-deriving any of it:

    from agents import AGENTS, session_cmd, harbor_kwargs, egress_hosts, usage

`egress_hosts()` is the union every staged task must allow, so a new task template cannot
silently omit a host and take a whole cohort down with it.
"""
from __future__ import annotations
import json, os, re, shlex, subprocess, glob

HOME = os.path.expanduser("~")
ENVFILE = "/home/evan/Documents/eval_tasks/.env"


def _envfile(key):
    """Read one var from the paid-key .env without importing it wholesale."""
    try:
        for ln in open(ENVFILE):
            m = re.match(rf"\s*(?:export\s+)?{re.escape(key)}=(.*)", ln)
            if m:
                return m.group(1).strip().strip('"').strip("'")
    except OSError:
        pass
    return None


def _devin_key():
    # harbor writes only windsurf_api_key into the container's credentials.toml
    try:
        for ln in open(f"{HOME}/.local/share/devin/credentials.toml"):
            if "windsurf_api_key" in ln:
                return ln.split('"')[1]
    except (OSError, IndexError):
        pass
    return None


def _grok_key():
    out = subprocess.run(["bash", "scripts/ops/grok_key.sh"], capture_output=True,
                         text=True, cwd="/home/evan/Documents/open_swe_traces_research")
    return (out.stdout or "").strip() or None


AGENTS = {
    "devin": {
        "harbor_agent": "devin",
        # the devin/ prefix is load-bearing: without it harbor's slug split yields a name
        # the CLI rejects as "Unknown model"
        "harbor_model": "devin/swe-2-max",
        "session_bin": "devin",
        "key": _devin_key,
        "key_env": "DEVIN_API_KEY",
        # reachable hosts REQUIRED for the in-container CLI to resolve its model list
        "hosts": ["api.devin.ai", "*.devin.ai", "app.devin.ai", "server.codeium.com",
                  "*.codeium.com", "*.cognition.ai", "*.windsurf.com"],
        "agent_env": {"DEVIN_API_SERVER_URL": "https://server.codeium.com"},
        "harbor_extra": ["--agent-timeout-multiplier", "4.0"],
        "needs_pty": False,
        "worktree_ok": True,
        "notes": "slowest; free. Trials ~20-40m. Cap counts sessions + in-container trials.",
    },
    "cursor": {
        "harbor_agent": "cursor-cli",
        "harbor_model": "cursor/composer-2.5",
        "session_bin": "cursor-agent",
        # the SESSION CLI wants the bare slug; harbor wants the prefixed one
        "session_model": "composer-2.5",
        "key": lambda: _envfile("CURSOR_API_KEY"),
        "key_env": "CURSOR_API_KEY",
        "hosts": ["cursor.com", "*.cursor.com", "*.cursor.sh", "downloads.cursor.com"],
        "agent_env": {},
        "harbor_extra": [],
        "needs_pty": False,
        "worktree_ok": True,
        # a stale free-plan key in the environment beats the .env one; unset it first
        "unset_env": ["CURSOR_API_KEY"],
        "notes": "fast, metered. Budget in outputs/composer_budget.json.",
    },
    "grok": {
        "harbor_agent": "grok-build",
        "harbor_model": "grok-4.6",
        "session_bin": "grok",
        "session_model": "grok-4.6",
        "key": _grok_key,
        "key_env": "XAI_API_KEY",
        "hosts": ["api.x.ai", "*.x.ai", "grok.com", "*.grok.com"],
        "agent_env": {},
        "harbor_extra": ["--ak", "reasoning_effort=high"],
        # without a PTY it dies with "No such device or address (os error 6)"; without a
        # stdin that never EOFs the job tears down as soon as the launcher returns
        "needs_pty": True,
        "worktree_ok": False,
        "notes": "HANGS in a git worktree - run in a plain directory and harvest after.",
    },
}


def egress_hosts(agents=None):
    """Union of hosts every staged task must allow. A task template that omits one of
    these takes down every trial for that agent with an error that looks unrelated."""
    out = []
    for name in (agents or AGENTS):
        for h in AGENTS[name]["hosts"]:
            if h not in out:
                out.append(h)
    return out


def harbor_kwargs(agent):
    """Args for `harbor run` for this agent, including the env it needs inside."""
    a = AGENTS[agent]
    kw = ["--agent", a["harbor_agent"], "--model", a["harbor_model"]]
    k = a["key"]()
    if k and a.get("key_env"):
        kw += ["--ae", f"{a['key_env']}={k}"]
    for ek, ev in a.get("agent_env", {}).items():
        kw += ["--ae", f"{ek}={ev}"]
    return kw + list(a.get("harbor_extra", []))


def session_cmd(agent, prompt, workdir, timeout=10800):
    """A full shell command running this agent headlessly against `prompt`."""
    a = AGENTS[agent]
    model = a.get("session_model", a["harbor_model"])
    if agent == "grok":
        inner = (f'{a["session_bin"]} --model {model} --always-approve '
                 f'--output-format streaming-json {shlex.quote(prompt)}')
        # sleep infinity keeps a writer inside the job's OWN process group, so the pipe
        # survives the launcher returning; script -qec supplies the PTY
        return (f'cd {shlex.quote(workdir)} && timeout {timeout} bash -c '
                f'{shlex.quote(f"sleep infinity | script -qec {shlex.quote(inner)} /dev/null")}')
    if agent == "cursor":
        unset = " ".join(f"-u {v}" for v in a.get("unset_env", []))
        return (f'env {unset} bash -c {shlex.quote(f"set -a; . {ENVFILE}; set +a; "
                f"cd {shlex.quote(workdir)}; exec timeout {timeout} "
                f"{a['session_bin']} --model {model} --force --print {shlex.quote(prompt)}")}')
    if agent == "devin":
        return (f'cd {shlex.quote(workdir)} && timeout {timeout} {a["session_bin"]} '
                f'--model swe-2-max --permission-mode dangerous --prompt-file {shlex.quote(prompt)}')
    raise ValueError(agent)


def usage(agent, workdir=None):
    """Tokens and USD for the most recent session of this agent, when it records them."""
    if agent == "grok":
        enc = (workdir or "").replace("/", "%2F")
        base = f"{HOME}/.grok/sessions/{enc}"
        cands = glob.glob(f"{base}/*/usage.json") or glob.glob(f"{HOME}/.grok/sessions/*/*/usage.json")
        if not cands:
            return None
        f = max(cands, key=os.path.getmtime)
        d = json.load(open(f)).get("session", {})
        return {"input": d.get("inputTokens"), "output": d.get("outputTokens"),
                "cached": d.get("cachedReadTokens"), "total": d.get("totalTokens"),
                "usd": d.get("costUsdTicks", 0) / 1e9, "calls": d.get("modelCalls")}
    if agent == "cursor":
        # composer spend is tracked against the budget file, not per session
        try:
            out = subprocess.run(["uv", "run", "python", "scripts/ops/composer_budget.py"],
                                 capture_output=True, text=True, timeout=120,
                                 cwd="/home/evan/Documents/open_swe_traces_research").stdout
            m = re.search(r"used ([\d.]+)M of ([\d.]+)M", out)
            return {"used_m": float(m.group(1)), "budget_m": float(m.group(2))} if m else None
        except Exception:
            return None
    return None


if __name__ == "__main__":
    import sys
    if len(sys.argv) > 1 and sys.argv[1] == "--hosts":
        print(json.dumps(egress_hosts()))
    else:
        for n, a in AGENTS.items():
            k = a["key"]()
            print(f"{n:8} bin={a['session_bin']:13} key={'OK' if k else 'MISSING':7} "
                  f"worktree={'yes' if a['worktree_ok'] else 'NO':3} pty={a['needs_pty']}")
            print(f"         {a['notes']}")
