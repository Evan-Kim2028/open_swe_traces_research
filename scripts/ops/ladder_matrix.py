#!/usr/bin/env python3
"""Second and third curves on units that already have a deep one — the comparison, not the count.

43 units have 5 or 6 of their 6 ladder cells measured by Composer. Almost none have a
second solver's curve, so the dataset can say "this unit needs L5" but not "needs L5 FOR
COMPOSER, and L2 for devin" — which is the claim the affordance ladder exists to support.
Only 2 units in the whole dataset have two independent curves.

The scarce resource is not trials, it is COMPLETED COMPARISONS. A half-finished ladder
compares nothing, so this finishes one (unit, solver) curve before starting the next rather
than spreading trials thin across many units. It also runs a curve's rungs CONCURRENTLY
where the guard allows, because at ~44 minutes a trial, six sequential rungs is most of a
day per solver per unit -- which is the actual reason second curves are so rare.

Two phases per (unit, solver), because the ladder has a real dependency and only one:

  phase 1   L0 and L2 together. A solver's higher rungs mean nothing until it has shown it
            fails the low ones itself; that is what makes the curves comparable rather than
            two unrelated facts. L2 needs SOME L0 on record, which the composer curve
            already provides, so these two run at once rather than in sequence.
  phase 2   L3, L4, L5, L6 together, only if the solver failed both. Rostered in
            ladder_backfill, which is what lets the guard take them out of order.

If the solver PASSES L2 the curve stops there: the flip is L2, everything above it is a
monotone-implied pass, and the comparison against composer's L5 is already complete. That
is the one place where stopping early costs nothing.

Free solvers only by default. devin and grok spend no budget, composer has 231M left and
already owns the baseline curve on every unit in the roster.

    ladder_matrix.py --plan              # roster and the work per unit
    ladder_matrix.py --run               # do it
    ladder_matrix.py --report            # curves side by side
"""
from __future__ import annotations

import argparse
import os
import pathlib
import subprocess
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
REPO = pathlib.Path(__file__).resolve().parents[2]
SWEEPS = REPO / "experiments" / "dose_response"
LADDER = ("0", "2", "3", "4", "5", "6")
PHASE1 = ("0", "2")
PHASE2 = ("3", "4", "5", "6")
COHORT_PREFIX = "sweep_matrix"
DEFAULT_SOLVERS = ("devin",)

# Grok is PINNED to the units it was pointed at: the three composer exhausted at L6
# (exprhash, httpencoding, httpmux), driven by grok_ladder.py. It is not a general-purpose
# second opinion here and must not be expanded to new tasks.
#
# This is a hard constraint in the tool rather than a note, because the roster this script
# writes is SOLVER-AGNOSTIC: rostering `archive 0` to let devin screen it also opens that
# cell to anything else that asks. Selecting grok for one new unit therefore leaks 55
# roster lines across 38 units in a single pass, which is exactly what happened and had to
# be unwound by hand while a grok trial was already running on `archive`.
GROK_PINNED = frozenset({"exprhash", "httpencoding", "httpmux"})
AGENT = {"devin": ("devin", "devin/swe-2-max"),
         "grok": ("grok-build", os.environ.get("GROK_MODEL", "grok-4.7")),
         "composer": ("cursor-cli", "composer-2.5")}
CONC = {"devin": 4, "grok": 3, "composer": 4}


def curves():
    import trial_ledger as TL
    return TL.ledger_by_solver()


def coverage(d) -> int:
    return sum(1 for r in LADDER if d.get(r))


def roster_units(bys, min_depth: int = 5) -> list[str]:
    """Units where SOME solver already has a deep curve to compare against.

    Derived rather than listed: a unit whose baseline is still being built is not yet a
    comparison candidate, and one that gains a second curve drops out on its own.
    """
    out = []
    for base, per in bys.items():
        depths = sorted((coverage(d) for d in per.values()), reverse=True)
        if depths and depths[0] >= min_depth and (len(depths) < 2 or depths[1] < 2):
            out.append(base)
    return sorted(out)


def flipped(d) -> bool:
    return any(max(v) > 0 for r, v in d.items()
               if r.isdigit() and int(r) >= 2 and v)


def open_cells(base: str, solver: str, per, bys) -> list[str]:
    """Rungs this solver may run right now, as the guard sees them."""
    import trial_guard as TG
    mine = (bys.get(base) or {}).get(solver) or {}
    if flipped(mine):
        return []          # flip found; higher rungs are monotone-implied
    want = PHASE1 if not (mine.get("0") and mine.get("2")) else PHASE2
    out = []
    for r in want:
        if mine.get(r):
            continue
        if staged_dir(base, r) is None:
            continue
        try:
            ok, _ = TG.decide(f"{base}-L{r}", per, solver=solver)
        except Exception:
            ok = False
        if ok:
            out.append(r)
    return out


def allowed(base: str, solver: str) -> bool:
    """Is this solver permitted on this unit at all, before any guard question."""
    if solver == "grok":
        return base in GROK_PINNED
    return True


def open_cells_rostered(base: str, solver: str, per, bys) -> list[str]:
    """open_cells, but with this unit's cells rostered FIRST.

    Several of the guard's gates -- "L0 already decided", the non-flip rule, the all-solver
    ceiling, and now out-of-order rungs -- are opened by the ladder_backfill roster, and
    they are all checked against the roster file on disk. Asking the guard before writing
    the roster therefore answers a question about the wrong world: it reported L0 closed on
    almost every unit here, which is exactly the cell a second solver most needs, since a
    curve with no L0 of its own cannot be compared to one that has it.

    Rostering is a statement of intent, not a side effect, so it is written only for cells
    that are STAGED and that this solver has no verdict for -- the same bound the guard
    applies when it honours them.
    """
    mine = (bys.get(base) or {}).get(solver) or {}
    if flipped(mine):
        return []
    want = PHASE1 if not (mine.get("0") and mine.get("2")) else PHASE2
    for r in want:
        if not mine.get(r) and staged_dir(base, r) is not None:
            roster(base, r)
    return open_cells(base, solver, per, bys)


def roster(base: str, rung: str) -> None:
    import trial_guard as TG
    p = REPO / TG.LADDER_BACKFILL_ROSTER
    p.parent.mkdir(parents=True, exist_ok=True)
    line = f"{base} {rung}"
    have = set(p.read_text().splitlines()) if p.is_file() else set()
    if line not in {h.strip() for h in have}:
        with open(p, "a") as fh:
            fh.write(line + "\n")


def staged_dir(base: str, rung: str) -> pathlib.Path | None:
    for h in sorted(SWEEPS.glob(f"sweep_*/{base}-L{rung}")):
        if (h / "instruction.md").is_file() and (h / "environment" / "src").is_dir():
            return h
    return None


def build_cohort(base: str, solver: str, rungs: list[str]) -> pathlib.Path | None:
    dest = SWEEPS / f"{COHORT_PREFIX}_{solver}_{base}"
    dest.mkdir(parents=True, exist_ok=True)
    made = 0
    for r in rungs:
        src = staged_dir(base, r)
        if src is None:
            continue
        link = dest / f"{base}-L{r}"
        if not link.exists():
            subprocess.run(["cp", "-al", str(src), str(link)], check=True)
        made += 1
    return dest if made else None


def launch(cohort: pathlib.Path, solver: str, n: int, log: pathlib.Path) -> int:
    agent, model = AGENT[solver]
    env = dict(os.environ, AGENT=agent, MODEL=model, GUARD_SOLVER=solver)
    if solver == "grok":
        env["GROK_EFFORT"] = os.environ.get("GROK_EFFORT", "high")
    cmd = ["bash", str(REPO / "scripts" / "ops" / "sweep_seq.sh"),
           cohort.name, "1", str(min(n, CONC[solver]))]
    with open(log, "ab") as fh:
        fh.write(f"\n=== {time.strftime('%H:%M:%S')} {solver} {cohort.name} "
                 f"({n} cell(s))\n".encode())
        fh.flush()
        return subprocess.run(cmd, cwd=REPO, env=env, stdout=fh, stderr=fh).returncode


def _live_bases() -> set[str]:
    try:
        out = subprocess.run(["docker", "ps", "--format", "{{.Names}}"],
                             capture_output=True, text=True, timeout=60).stdout
    except Exception:
        return set()
    live = set()
    for n in out.split():
        if "__env-main" in n:
            stem = n.split("__", 1)[0]
            if "-l" in stem:
                live.add(stem.rsplit("-l", 1)[0])
    return live


def plan(solvers):
    import trial_ledger as TL
    bys = curves()
    per = TL.ledger()
    units = roster_units(bys)
    rows = []
    for base in units:
        for s in solvers:
            if not allowed(base, s):
                continue
            cells = open_cells_rostered(base, s, per, bys)
            if cells:
                rows.append((base, s, cells))
    return units, rows


def cmd_plan(solvers) -> int:
    units, rows = plan(solvers)
    print(f"comparison candidates (deep curve from one solver, none from another): {len(units)}")
    print(f"(unit, solver) curves with work available: {len(rows)}")
    tot = 0
    for base, s, cells in rows[:30]:
        tot += len(cells)
        ph = "phase1 L0+L2" if set(cells) & set(PHASE1) else "phase2 L3-L6"
        print(f"  {base:22s} {s:7s} {ph:13s} rungs {','.join('L'+c for c in cells)}")
    print(f"trials in the first {min(30, len(rows))} curve(s): {tot}")
    return 0


def cmd_report(solvers) -> int:
    bys = curves()
    shown = 0
    for base in sorted(bys):
        per = bys[base]
        deep = [s for s, d in per.items() if coverage(d) >= 2]
        if len(deep) < 2:
            continue
        shown += 1
        print(f"\n{base}")
        for s in sorted(per):
            d = per[s]
            if not d:
                continue
            trail = "  ".join(f"L{r}:" + "/".join("P" if x > 0 else "f" for x in d[r])
                              for r in sorted(d, key=lambda z: int(z) if z.isdigit() else 9))
            fl = [int(r) for r, v in d.items()
                  if r.isdigit() and int(r) >= 2 and v and max(v) > 0]
            v = f"flips L{min(fl)}" if fl else f"no flip in {coverage(d)} cell(s)"
            print(f"  {s:9s} {trail}   -> {v}")
    print(f"\nunits with two or more solver curves: {shown}")
    return 0


def cmd_run(solvers, max_curves: int) -> int:
    import trial_ledger as TL
    log = REPO / "outputs" / "supervisor" / "ladder_matrix.log"
    log.parent.mkdir(parents=True, exist_ok=True)
    done = 0
    while done < max_curves:
        bys = curves()
        per = TL.ledger()
        picked = None
        for base in roster_units(bys):
            if base in _live_bases():
                continue
            for s in solvers:
                if not allowed(base, s):
                    continue
                cells = open_cells_rostered(base, s, per, bys)
                if cells:
                    picked = (base, s, cells)
                    break
            if picked:
                break
        if picked is None:
            print("no curve has open work right now", flush=True)
            return cmd_report(solvers)
        base, s, cells = picked
        print(f"[{done+1}] {base} / {s}: rungs {','.join('L'+c for c in cells)}", flush=True)
        for r in cells:
            roster(base, r)
        cohort = build_cohort(base, s, cells)
        if cohort is None:
            print(f"  nothing stageable for {base}; skipping", flush=True)
            done += 1
            continue
        rc = launch(cohort, s, len(cells), log)
        after = curves()
        got = (after.get(base) or {}).get(s) or {}
        trail = "  ".join(f"L{r}:" + "/".join("P" if x > 0 else "f" for x in got[r])
                          for r in sorted(got, key=lambda z: int(z) if z.isdigit() else 9))
        print(f"  rc={rc}  {s} now: {trail or '(no verdict)'}", flush=True)
        done += 1
    return cmd_report(solvers)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--plan", action="store_true")
    ap.add_argument("--run", action="store_true")
    ap.add_argument("--report", action="store_true")
    ap.add_argument("--solvers", default=",".join(DEFAULT_SOLVERS))
    ap.add_argument("--max-curves", type=int, default=40)
    a = ap.parse_args()
    solvers = tuple(x.strip() for x in a.solvers.split(",") if x.strip())
    if a.report:
        return cmd_report(solvers)
    if a.run:
        return cmd_run(solvers, a.max_curves)
    return cmd_plan(solvers)


if __name__ == "__main__":
    raise SystemExit(main())
