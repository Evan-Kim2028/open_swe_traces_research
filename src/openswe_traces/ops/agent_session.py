#!/usr/bin/env python3
"""Run one headless agent session against a brief. The only launcher.

Replaces grok_job.sh and cursor_job.sh, which each re-derived the same things and drifted:
the PTY/stdin dance, which key file to read, which model slug each CLI accepts, which env
must be UNSET. All of that now lives in agents.py.

  agent_session.py <agent> <job-name> <brief-path> <workdir> [timeout_sec]

Refuses to run grok in a git worktree, where it hangs with 148 bytes of terminal init and
no deliverable - a failure that looks like the model is down.
"""
import os, subprocess, sys

from openswe_traces.ops.agents import AGENTS, session_cmd, usage


def is_worktree(d):
    p = os.path.join(d, ".git")
    if os.path.isfile(p):                       # a worktree's .git is a FILE, not a dir
        try:
            return "gitdir:" in open(p).read()
        except OSError:
            return False
    return False


def main():
    if len(sys.argv) < 5:
        print(__doc__)
        return 2
    agent, job, brief, wd = sys.argv[1:5]
    timeout = int(sys.argv[5]) if len(sys.argv) > 5 else 10800
    if agent not in AGENTS:
        print(f"unknown agent {agent}; have {', '.join(AGENTS)}")
        return 2
    if not AGENTS[agent]["worktree_ok"] and is_worktree(wd):
        print(f"REFUSED: {agent} hangs in a git worktree ({wd}). "
              f"Extract the repo into a plain directory and harvest afterwards.")
        return 3
    os.makedirs(os.path.join(wd, "outputs"), exist_ok=True)
    log = os.path.join(wd, "outputs", f"{job}.{agent}.log")

    if agent == "devin":
        prompt = brief                          # devin takes a --prompt-file
    else:
        prompt = (f"Read the file {brief} and carry out the job it describes, in full, "
                  f"in this working directory. Follow every rule in it. When the "
                  f"deliverable is written, stop.")
    cmd = session_cmd(agent, prompt, wd, timeout)
    print(f"{agent}: {job} -> {log}")
    with open(log, "wb") as fh:
        rc = subprocess.run(cmd, shell=True, stdout=fh, stderr=subprocess.STDOUT).returncode
    u = usage(agent, wd)
    if u:
        print(f"  usage: {u}")
    print(f"  exit {rc}, log {os.path.getsize(log)}B")
    return rc


def cli():
    sys.exit(main())


if __name__ == "__main__":
    cli()
