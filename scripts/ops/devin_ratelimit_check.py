#!/usr/bin/env python3
"""Detect Devin throttling early, instead of inferring it from a session going quiet.

A rate limit is usually cumulative, so "no 429 in the first hour" says little about hour
three. This looks for the two signatures that actually show up: an explicit 429/quota
message in a session's transcript, and sessions whose activity stalls together (a shared
limit throttles every session at once, unlike one agent getting stuck).
"""
import sqlite3, os, re, time, glob, sys

DB = os.path.expanduser("~/.local/share/devin/cli/sessions.db")
# Only error-SHAPED signatures. A bare "rate limit" also matches a Go test named
# TestRateLimits and a skill description; the first version of this fired on all three.
# A throttle shows up as an HTTP status or an API error string, not as prose.
PAT = re.compile(
    # The real Devin message, copied verbatim from a session that died on it. Three
    # earlier versions of this regex were written from guesses and matched none of it:
    #   "Reached free model rate limit. Upgrade to Max for higher limits ...
    #    Your limit will reset in 6 minutes."
    r"reached\s+(?:the\s+)?free\s+model\s+rate\s+limit"
    r"|your\s+limit\s+will\s+reset\s+in"
    r"|upgrade\s+to\s+max\s+for\s+higher\s+limits"
    r"|errorKind[\"']?\s*[:=]\s*[\"']?unavailable"
    # generic HTTP shapes, still useful for other providers
    r"|\b429\s+too\s+many\s+requests\b"
    r"|\bHTTP[/ ]?\d?\.?\d?\s*429\b"
    r"|[\"']?status(?:_code)?[\"']?\s*[:=]\s*429\b"
    r"|\brate[ _-]?limit(?:ed)?\s+exceeded\b"
    r"|\bquota\s+exceeded\b",
    re.I)


def scan_cli_logs():
    """Scan the CLI's OWN stdout/stderr, not the agent transcript.

    Transcript matching does not work in this corpus: the go-github agent reads
    go-github's rate-limit handling, so "Retry-After" and "429" appear as source code
    being discussed. A real throttle is reported by the CLI itself, so only its logs count.
    """
    hits = []
    for wt in glob.glob("/home/evan/Documents/oswt-*/outputs/*.log"):
        try:
            if time.time() - os.path.getmtime(wt) > 6 * 3600:
                continue
            with open(wt, errors="replace") as fh:
                fh.seek(max(0, os.path.getsize(wt) - 200_000))
                for ln in fh:
                    m = PAT.search(ln)
                    if m:
                        hits.append((os.path.basename(wt), m.group(0), " ".join(ln.split())[:150]))
                        break
        except Exception:
            pass
    return hits


def main():
    if not os.path.exists(DB):
        print("no sessions.db"); return 0
    c = sqlite3.connect(f"file:{DB}?mode=ro", uri=True)
    now = time.time()
    rows = c.execute("""select id, working_directory, last_activity_at
                        from sessions order by created_at desc limit 8""").fetchall()
    live, seen_wd = [], {}
    hits = scan_cli_logs()
    for sid, wd, la in rows:
        try:
            lav = float(la); lav = lav / 1000 if lav > 1e12 else lav
        except Exception:
            continue
        idle = (now - lav) / 60
        if idle > 90:                       # not a current session
            continue
        key = os.path.basename(wd or "?")
        # sessions.db keeps a row per restart; keep only the freshest per worktree so the
        # count matches running processes rather than history
        if key not in seen_wd or idle < seen_wd[key]:
            seen_wd[key] = idle
    live = sorted(seen_wd.items(), key=lambda kv: kv[1])
    print(f"live sessions: {len(live)}")
    for wd, idle in live:
        print(f"  {wd:18s} idle {idle:5.1f}m")
    if hits:
        print("\nTHROTTLE SIGNATURES:")
        for wd, tok, ctx in hits:
            print(f"  {wd}: matched {tok!r}")
            print(f"    …{' '.join(ctx.split())}…")
    else:
        print("\nno throttle signature in recent messages")

    # A shared limit stalls everything at once; one stuck agent does not.
    if len(live) >= 3:
        stalled = [w for w, i in live if i > 10]
        if len(stalled) == len(live):
            print(f"\nWARNING: all {len(live)} sessions idle >10m simultaneously — "
                  f"consistent with a shared limit, not one stuck agent")
            return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
