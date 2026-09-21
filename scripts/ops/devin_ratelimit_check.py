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



SAFE_CAP = 4
OVERRIDE = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(
    os.path.abspath(__file__)))), "outputs", "supervisor", "devin_cap_override")


def drop_cap(why: str) -> None:
    """Pull Devin concurrency back to a level that has never been throttled.

    Printing a warning is not a response. A throttle errors every in-flight trial, so by
    the time anyone reads the log the work is already lost — and the next tick launches
    straight back into the limit. slots.CAP reads this file, so writing it takes effect
    on the very next launch with nothing to restart.

    Deliberately one-way: this lowers, never raises. Restoring the cap is a human
    decision, made after looking at why the limit was hit, and is done by deleting the
    file (or exporting DEVIN_CAP, which outranks it).
    """
    try:
        cur = int(open(OVERRIDE).read().split()[0])
    except (OSError, ValueError, IndexError):
        cur = None
    if cur is not None and cur <= SAFE_CAP:
        return
    os.makedirs(os.path.dirname(OVERRIDE), exist_ok=True)
    with open(OVERRIDE, "w") as fh:
        fh.write(f"{SAFE_CAP}\n# dropped {time.strftime('%Y-%m-%dT%H:%M:%S')}: {why}\n"
                 f"# delete this file to restore the default cap\n")
    print(f"CAP DROPPED to {SAFE_CAP} — {why}")


def error_rate_check(window_min: int = 45, floor: int = 4, pct: float = 0.25) -> int:
    """Drop the cap when recent Devin trials start FAILING, not only when a limit says so.

    The throttle regex needs Devin to say the words. A limit reached mid-trial does not
    always announce itself in a transcript we can read — it shows up as trials erroring
    out, which is what a timeout looks like from our side too. Either way the response is
    the same: fewer slots.

    Deliberately requires a FLOOR of trials before acting. One error in one trial is
    noise, and dropping the cap on noise costs throughput for nothing. The reaper bug
    produced a 14% error rate over 36 trials; the clean run since has been 0% over 31,
    so 25% of at least 4 recent trials is well clear of both.
    """
    try:
        sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
        import trial_ledger as TL
    except Exception:
        return 0
    cutoff = time.time() - window_min * 60
    recent = [t for t in TL.trials()
              if t.get("model") == "swe-2-max" and t["mtime"] >= cutoff]
    if len(recent) < floor:
        print(f"devin: {len(recent)} trial(s) in {window_min}m — too few to judge")
        return 0
    bad = sum(1 for t in recent if t["errored"])
    rate = bad / len(recent)
    print(f"devin: {bad}/{len(recent)} recent trials errored ({rate:.0%}) in {window_min}m")
    if rate >= pct:
        drop_cap(f"{bad} of {len(recent)} trials errored in {window_min}m ({rate:.0%})")
        return 2
    return 0

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
        # An explicit limit message is the strongest signal there is; act on it too,
        # not only on the weaker all-sessions-idle heuristic below.
        drop_cap(f"explicit throttle signature in {len(hits)} session(s)")
    else:
        print("\nno throttle signature in recent messages")

    rc = error_rate_check()

    # A shared limit stalls everything at once; one stuck agent does not.
    if len(live) >= 3:
        stalled = [w for w, i in live if i > 10]
        if len(stalled) == len(live):
            print(f"\nWARNING: all {len(live)} sessions idle >10m simultaneously — "
                  f"consistent with a shared limit, not one stuck agent")
            drop_cap(f"all {len(live)} sessions idle >10m together")
            return 2
    return rc


if __name__ == "__main__":
    sys.exit(main())
