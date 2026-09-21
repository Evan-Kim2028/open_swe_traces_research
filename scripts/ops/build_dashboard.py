#!/usr/bin/env python3
"""Regenerate the task foundry dashboard from the live ledger.

The dataset count is a vanity number: it only goes up. The number that says whether a
decision worked is MARGINAL trials-per-certified — how many trials the last batch of
tasks cost. Decisions are drawn as vertical markers so a change that did nothing is
as visible as one that worked.

Data is inlined rather than fetched, because file:// cannot read a sibling JSON.
"""
import os
import json, os, subprocess, time

HIST = "outputs/history.json"
OUT = "analytics/research/dashboard.html"

# Pinned, not taken from mtime: editing a script today would otherwise redate the
# decision it implements. Sources are in the commit message and the session record.
DECISIONS = [
    ("2026-09-20T13:21", "B9 hack audit on every sweep"),
    ("2026-09-20T14:12", "free deterministic linter"),
    ("2026-09-20T14:34", "sequential escalation (k=1)"),
    ("2026-09-20T15:33", "contract gap-read pre-trial"),
    ("2026-09-20T17:31", "trial guard"),
    ("2026-09-20T17:37", "continuous supervisor"),
]


def sh(c, d=""):
    try:
        return subprocess.run(c, shell=True, capture_output=True, text=True, timeout=30).stdout.strip() or d
    except Exception:
        return d


def ts(iso):
    return time.mktime(time.strptime(iso, "%Y-%m-%dT%H:%M"))


def main():
    subprocess.run(["python3", "scripts/ops/backfill_history.py"], capture_output=True)
    h = json.load(open(HIST))
    rows = h["rows"]
    if not rows:
        print("no history")
        return 1

    last = rows[-1]
    now = time.time()
    r6 = [r for r in rows if r["t"] >= now - 6 * 3600]
    d6c = (r6[-1]["certified"] - r6[0]["certified"]) if len(r6) > 1 else 0
    d6t = (r6[-1]["trials_cum"] - r6[0]["trials_cum"]) if len(r6) > 1 else 0
    tpc6 = round(d6t / d6c, 1) if d6c > 0 else None

    # Count authored units across the main checkout AND every worktree, deduplicated.
    # A main-only count read 144 where the real figure is ~507: batch2, batch4 and all
    # seven au5 batches were authored inside worktrees and never merged. Under-reporting
    # the numerator hid the fact that authoring is far ahead of verification.
    import glob as _g
    import sys as _sys
    _sys.path.insert(0, "scripts/ops")
    try:
        import trial_guard as _tg
        per = _tg.ledger()
    except Exception:
        per = {}
    _seen = {}
    for _root in ["."] + sorted(_g.glob("/home/evan/Documents/oswt-*")):
        for _d in _g.glob(f"{_root}/experiments/pipeline/authored*/*/*/"):
            if not os.path.isdir(os.path.join(_d, "_author")):
                continue
            _p = _d.rstrip("/").split("/")
            _batch, _repo, _unit = _p[-3], _p[-2], _p[-1]
            if _repo.startswith("_"):
                continue
            # The au5 units were consolidated into authored_batch5, so each exists under
            # two batch names and was counted twice (629 against a true 507). Fold the
            # per-repo au5 batches onto their consolidated name before deduplicating.
            if _batch.startswith("authored_au5"):
                _batch = "authored_batch5"
            _suite = bool(_g.glob(_d + "tests/hidden/**/*_test.go", recursive=True))
            _con = os.path.exists(os.path.join(_d, "_author/contract.md"))
            _o = _seen.get((_batch, _repo, _unit), (False, False))
            _seen[(_batch, _repo, _unit)] = (_o[0] or _suite, _o[1] or _con)
    authored = len(_seen)
    # A unit that has been trialled was demonstrably verified, even when its suite lives
    # in the staged sweep dir rather than _author/tests (the older batch2 path). Without
    # this the funnel showed 316 trialled against 138 verified - not a funnel at all.
    try:
        _trialled_names = {u.split("-", 1)[-1] for u in per} | set(per)
    except Exception:
        _trialled_names = set()

    def _was_verified(key, v):
        return v[0] or key[2] in _trialled_names or f"{key[1]}-{key[2]}" in _trialled_names

    verified = sum(1 for k, v in _seen.items() if _was_verified(k, v))
    stageable = sum(1 for k, v in _seen.items() if _was_verified(k, v) and v[1])
    unverified = authored - verified
    nullified = len(_g.glob("experiments/dose_response/_nullified/*/*/"))
    easy = sum(1 for _ in ())  # filled below from guard
    try:
        import sys
        sys.path.insert(0, "scripts/ops")
        import trial_guard as g
        per = g.ledger()
        easy = len([u for u, d in per.items() if d.get("0") and max(d["0"]) > 0])
        nonflip = len([u for u, d in per.items() if d.get("0") and max(d["0"]) == 0
                       and len(d.get("2", [])) >= g.NONFLIP_CAP and max(d.get("2") or [1]) == 0])
    except Exception:
        nonflip = 0
    free = sh("df --output=avail -BG / | tail -1", "?G").strip()
    # A Devin TRIAL runs the CLI inside a container: invisible to pgrep but it occupies a
    # slot exactly as a session does. Counting sessions alone displayed 2/4 while the real
    # figure was 4/4, which is the miscount that let the cap drift to 7 unnoticed.
    _ds = int(sh("pgrep -af '[d]evin --model' | grep -oP 'closure_\\w+' | sort -u | wc -l", "0") or 0)
    _dt = int(sh("ps -eo args | awk '/harbor run/ && !/awk/ {a=\"\";n=1;j=\"\"; "
                 "for(i=1;i<NF;i++){if($i==\"--agent\")a=$(i+1); "
                 "if($i==\"--n-concurrent\")n=$(i+1); if($i==\"--job-name\")j=$(i+1)} "
                 "if(a==\"devin\" && !(j in s)){s[j]=1; t+=n}} END{print t+0}'", "0") or 0)
    devin = _ds + _dt
    conts = sh("docker ps --format '{{.Names}}' | grep -c env-main", "0")

    # ---- geometry ----
    W, H, PL, PR, PT, PB = 1120, 300, 86, 150, 26, 62
    t0, t1 = rows[0]["t"], max(rows[-1]["t"], now)
    span = max(t1 - t0, 3600)

    def X(t):
        return PL + (t - t0) / span * (W - PL - PR)

    def axis_x(yb):
        out, step = [], 3 * 3600
        tt = int(t0 // step) * step
        while tt <= t1:
            if tt >= t0:
                x = X(tt)
                lab = time.strftime("%a %H:%M", time.localtime(tt))
                out.append(f'<line x1="{x:.1f}" y1="{yb}" x2="{x:.1f}" y2="{yb+5}" class="tick"/>'
                           f'<text x="{x:.1f}" y="{yb+20}" class="axl" text-anchor="middle">{lab}</text>')
            tt += step
        return "".join(out)

    def marks(ytop, ybot):
        out = []
        for i, (iso, lab) in enumerate(DECISIONS):
            t = ts(iso)
            if not (t0 <= t <= t1):
                continue
            x = X(t)
            dy = ytop + 12 + (i % 3) * 15
            out.append(f'<line x1="{x:.1f}" y1="{ytop}" x2="{x:.1f}" y2="{ybot}" class="mk"/>'
                       f'<text x="{x+4:.1f}" y="{dy}" class="mkl">{lab}</text>')
        return "".join(out)

    # ---- chart 1: the dataset ----
    cmax = max(r["certified"] for r in rows) or 1
    tmax = max(r["trials_cum"] for r in rows) or 1
    yb = H - PB

    def Yc(v): return yb - v / cmax * (yb - PT)
    def Yt(v): return yb - v / tmax * (yb - PT)

    cpts = " ".join(f"{X(r['t']):.1f},{Yc(r['certified']):.1f}" for r in rows)
    tpts = " ".join(f"{X(r['t']):.1f},{Yt(r['trials_cum']):.1f}" for r in rows)
    g1 = "".join(f'<line x1="{PL}" y1="{Yc(v):.1f}" x2="{W-PR}" y2="{Yc(v):.1f}" class="grid"/>'
                 f'<text x="{PL-10}" y="{Yc(v)+5:.1f}" class="axl" text-anchor="end">{v}</text>'
                 for v in range(0, cmax + 1, max(1, cmax // 4)))
    g1r = "".join(f'<text x="{W-PR+10}" y="{Yt(v)+5:.1f}" class="axl axr">{v}</text>'
                  for v in range(0, tmax + 1, max(1, tmax // 4)))

    # ---- chart 2: the payoff ----
    H2 = 300
    prod = [r for r in rows if r["tpc_marginal"]]
    mmax = max((r["tpc_marginal"] for r in prod), default=1)
    yb2 = H2 - PB

    def Ym(v): return yb2 - min(v, mmax) / mmax * (yb2 - PT)

    bw = max(6, (W - PL - PR) / max(len(rows), 1) * 0.66)
    bars = "".join(
        f'<rect x="{X(r["t"])-bw/2:.1f}" y="{Ym(r["tpc_marginal"]):.1f}" width="{bw:.1f}" '
        f'height="{yb2-Ym(r["tpc_marginal"]):.1f}" class="bar"/>'
        f'<text x="{X(r["t"]):.1f}" y="{Ym(r["tpc_marginal"])-6:.1f}" class="bl" text-anchor="middle">'
        f'{r["tpc_marginal"]:.0f}</text>' for r in prod)
    floor_y = Ym(2.6)
    g2 = "".join(f'<line x1="{PL}" y1="{Ym(v):.1f}" x2="{W-PR}" y2="{Ym(v):.1f}" class="grid"/>'
                 f'<text x="{PL-10}" y="{Ym(v)+5:.1f}" class="axl" text-anchor="end">{v:.0f}</text>'
                 for v in [0, mmax * .25, mmax * .5, mmax * .75, mmax])

    def tile(v, l, s=""):
        return (f'<div class="t"><div class="v">{v}</div><div class="l">{l}</div>'
                f'{f"<div class=s>{s}</div>" if s else ""}</div>')

    tiles = "".join([
        tile(last["certified"], "certified units",
             "fails L0, passes at L2 or above"),
        tile(last["trials_cum"], "trials run", "2.08M tokens each"),
        tile(last["tpc_cum"], "trials / certified", "cumulative · floor 2.6"),
        tile(tpc6 if tpc6 else "—", "trials / certified", "last 6h · marginal"),
        tile(easy, "killed at L0", "not hard enough"),
        tile(nonflip, "non-flipping", "need contract repair"),
        tile(f"{devin}/4", "Devin runs", f"sessions + in-container trials"),
        tile(free, "disk free", "prune floor 100G"),
    ])

    # ---- the funnel ------------------------------------------------------------------
    # Authoring ran far ahead of verification all night and nothing showed it: an authored
    # unit with no hidden suite cannot be staged, trialled or certified, so it is not an
    # asset. 507 authored against 138 verified is the single most decision-relevant number
    # on this page, and it was the one number the dashboard never had.
    trialled = len(per) if per else 0
    stages = [("authored", authored, "unit cut from a repo"),
              ("verified", verified, "hidden suite written"),
              ("stageable", stageable, "+ contract, can reach L2"),
              ("trialled", trialled, "at least one verdict"),
              ("certified", last["certified"], "fails L0, passes at L2 or above")]
    fmax = max(x[1] for x in stages) or 1
    funnel = ""
    for i, (name, n, sub) in enumerate(stages):
        pct = n / fmax * 100
        drop = ""
        if i:
            prev = stages[i - 1][1]
            lost = prev - n
            if lost > 0:
                drop = f'<span class="drop">&minus;{lost:,}</span>'
        funnel += (f'<div class="frow"><div class="fname">{name}{drop}</div>'
                   f'<div class="fbar"><span style="width:{pct:.1f}%"></span></div>'
                   f'<div class="fnum">{n:,}</div>'
                   f'<div class="fsub">{sub}</div></div>')
    funnel_card = (f'<div class="card"><h2>Where units are lost</h2>'
                   f'<p class="note">Every stage an authored unit must clear before it '
                   f'counts. {unverified:,} authored units have no hidden suite yet &mdash; '
                   f'they cannot be staged or certified at any rung, and verification is '
                   f'the only stage that converts them.'
                   + (f' {nullified:,} trials were nullified as infrastructure failures '
                      f'(agent never ran) and are excluded throughout.' if nullified else '')
                   + f'</p><div class="funnel">{funnel}</div></div>')

    html = f"""<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Task Foundry</title>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@500;600&family=IBM+Plex+Sans:wght@400;500;600;700&display=swap">
<style>
:root{{--bg:#F1F4F1;--card:#FBFCFA;--ink:#16211F;--ink2:#63746F;--rule:#C6D1CC;
 --acc:#0B6E5F;--acc2:#2E4F8F;--warn:#AB3E27;--bar:#2E4F8F;}}
@media(prefers-color-scheme:dark){{:root:not([data-theme="light"]){{--bg:#101614;--card:#1B2321;
 --ink:#E8EFEB;--ink2:#93A6A0;--rule:#33423E;--acc:#4FC0A8;--acc2:#7FA3E8;--warn:#E1755A;--bar:#7FA3E8;}}}}
:root[data-theme="dark"]{{--bg:#101614;--card:#1B2321;--ink:#E8EFEB;--ink2:#93A6A0;
 --rule:#33423E;--acc:#4FC0A8;--acc2:#7FA3E8;--warn:#E1755A;--bar:#7FA3E8;}}
*{{box-sizing:border-box}}
body{{margin:0;background:var(--bg);color:var(--ink);font-family:"IBM Plex Sans",system-ui,sans-serif}}
.wrap{{max-width:1180px;margin:0 auto;padding:28px 16px 48px}}
h1{{font-size:27px;font-weight:700;margin:0 0 4px;letter-spacing:-.02em}}
.sub{{color:var(--ink2);font-size:15px;margin:0 0 24px}}
.tiles{{display:grid;grid-template-columns:repeat(auto-fit,minmax(168px,1fr));gap:12px;margin-bottom:30px}}
.t{{background:var(--card);border:1px solid var(--rule);border-radius:7px;padding:15px 16px}}
.v{{font-family:"IBM Plex Mono",monospace;font-size:31px;font-weight:600;letter-spacing:-.02em;line-height:1.05}}
.l{{font-size:14px;font-weight:600;margin-top:5px}}
.s{{font-size:12.5px;color:var(--ink2);margin-top:3px}}
.card{{background:var(--card);border:1px solid var(--rule);border-radius:7px;padding:18px 18px 8px;margin-bottom:22px}}
h2{{font-size:18px;font-weight:600;margin:0 0 3px}}
.note{{font-size:13.5px;color:var(--ink2);margin:0 0 10px;max-width:74ch;line-height:1.5}}
.funnel{{margin:14px 0 16px}}
.frow{{display:grid;grid-template-columns:132px 1fr 62px 200px;gap:12px;align-items:center;
 padding:7px 0;border-top:1px solid var(--rule)}}
.frow:first-child{{border-top:none}}
.fname{{font-size:13.5px;font-weight:600}}
.drop{{font-family:"IBM Plex Mono",monospace;font-size:11px;color:var(--warn);margin-left:6px;font-weight:500}}
.fbar{{height:13px;background:color-mix(in srgb,var(--rule) 45%,transparent);border-radius:3px;overflow:hidden}}
.fbar span{{display:block;height:100%;background:var(--acc);border-radius:3px}}
.fnum{{font-family:"IBM Plex Mono",monospace;font-size:15px;font-weight:600;text-align:right}}
.fsub{{font-size:12px;color:var(--ink2)}}
@media(max-width:720px){{.frow{{grid-template-columns:104px 1fr 54px;row-gap:2px}}
 .fsub{{grid-column:1/-1;padding-bottom:2px}}}}
.scroll{{width:100%;overflow-x:auto}} svg{{display:block;min-width:900px;width:100%;height:auto}}
text{{font-family:"IBM Plex Sans",sans-serif;fill:var(--ink)}}
.axl{{font-size:12.5px;fill:var(--ink2)}} .axr{{fill:var(--acc2)}}
.axt{{font-size:13.5px;font-weight:600}}
.grid{{stroke:var(--rule);stroke-width:1;opacity:.5}} .tick{{stroke:var(--rule);stroke-width:1.5}}
.mk{{stroke:var(--warn);stroke-width:1.5;stroke-dasharray:4 4;opacity:.75}}
.mkl{{font-size:11.5px;fill:var(--warn);font-weight:600}}
.bar{{fill:var(--bar);opacity:.82}} .bl{{font-size:11px;fill:var(--ink2)}}
.lc{{fill:none;stroke:var(--acc);stroke-width:3}} .lt{{fill:none;stroke:var(--acc2);stroke-width:2;stroke-dasharray:5 4}}
.floor{{stroke:var(--acc);stroke-width:2;stroke-dasharray:7 4}}
.leg{{font-size:13px;color:var(--ink2);margin:8px 0 12px;display:flex;gap:20px;flex-wrap:wrap}}
.sw{{display:inline-block;width:15px;height:3px;vertical-align:middle;margin-right:6px}}
footer{{color:var(--ink2);font-size:12.5px;margin-top:8px}}
</style></head><body><div class="wrap">
<h1>Task Foundry</h1>
<p class="sub">Mining verifiable SWE tasks with controlled difficulty · generated {time.strftime('%a %d %b %H:%M')}</p>
<div class="tiles">{tiles}</div>
{funnel_card}

<div class="card"><h2>The dataset, and what it cost</h2>
<p class="note">A unit is certified when it fails at L0 and passes at L2 <em>or higher</em> — hard, fair, solvable and verifiable.
The rung it needs is its difficulty: a unit that only flips at L5 is harder than one that flips at L2, not a failed task.
Certified units are the product; trials are the bill.</p>
<div class="leg"><span><span class="sw" style="background:var(--acc)"></span>certified units (left)</span>
<span><span class="sw" style="background:var(--acc2)"></span>cumulative trials (right)</span>
<span><span class="sw" style="background:var(--warn)"></span>decision landed</span></div>
<div class="scroll"><svg viewBox="0 0 {W} {H}" role="img" aria-label="Certified units and cumulative trials over time">
{g1}{g1r}<line x1="{PL}" y1="{yb}" x2="{W-PR}" y2="{yb}" class="tick"/>
<line x1="{PL}" y1="{PT}" x2="{PL}" y2="{yb}" class="tick"/>{axis_x(yb)}
<polyline points="{tpts}" class="lt"/><polyline points="{cpts}" class="lc"/>{marks(PT, yb)}
<text x="18" y="{PT+8}" class="axt" fill="var(--acc)" transform="rotate(-90 18 {PT+8})" text-anchor="end">certified units</text>
<text x="{W-24}" y="{PT+8}" class="axt" fill="var(--acc2)" transform="rotate(-90 {W-24} {PT+8})" text-anchor="end">cumulative trials</text>
<text x="{(PL+W-PR)/2}" y="{H-12}" class="axt" text-anchor="middle">time</text></svg></div></div>

<div class="card"><h2>Does each decision make a task cheaper to mine?</h2>
<p class="note">Marginal cost: trials spent per unit certified <em>in that hour</em>. The cumulative average
({last['tpc_cum']}) drags a whole day of history behind it and cannot show a change working. The green line is the
theoretical floor of 2.6 — one L0 trial to prove hard, one L2 to prove solvable, plus screening overhead.
Bars above the floor are the waste still being paid.</p>
<div class="scroll"><svg viewBox="0 0 {W} {H2}" role="img" aria-label="Marginal trials per certified unit per hour">
{g2}<line x1="{PL}" y1="{yb2}" x2="{W-PR}" y2="{yb2}" class="tick"/>
<line x1="{PL}" y1="{PT}" x2="{PL}" y2="{yb2}" class="tick"/>{axis_x(yb2)}{bars}
<line x1="{PL}" y1="{floor_y:.1f}" x2="{W-PR}" y2="{floor_y:.1f}" class="floor"/>
<text x="{W-PR+8}" y="{floor_y+4:.1f}" class="axl" fill="var(--acc)">floor 2.6</text>
{marks(PT, yb2)}
<text x="18" y="{PT+8}" class="axt" transform="rotate(-90 18 {PT+8})" text-anchor="end">trials per certified unit</text>
<text x="{(PL+W-PR)/2}" y="{H2-12}" class="axt" text-anchor="middle">time</text></svg></div></div>

<footer>Sources: every reward.txt and result.json under experiments/dose_response/jobs, replayed in
mtime order by backfill_history.py. Decision times are pinned, not read from file mtimes, because editing a
script would otherwise redate the decision it implements. Regenerated each supervisor tick.</footer>
</div></body></html>"""
    open(OUT, "w").write(html)
    print(f"wrote {OUT} ({len(html)//1024}KB) — certified {last['certified']}, "
          f"cum {last['tpc_cum']} tpc, last-6h marginal {tpc6}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
