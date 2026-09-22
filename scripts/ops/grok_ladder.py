#!/usr/bin/env python3
"""Run a second model's COMPLETE affordance ladder on the units the first model exhausted.

Composer takes three units to L6 and fails every rung: exprhash, httpencoding, httpmux.
Those are the only units in the dataset where the ladder ran out without a flip, so they
are the ones worth pointing a different model at — either grok flips one, which makes it a
per-solver difficulty result of the strongest kind available (one model needs more than the
ladder has, another does not), or grok exhausts them too, which is much better evidence
that the unit is hard than one model's opinion.

FULL ladder, every rung, not "stop at the flip". Monotonicity says a solver that passes L3
would also pass L4, so the higher cells look inferable — but a dose-response curve whose
cells are inferred cannot test the assumption it was inferred from. The cells are measured.

Resumable and idempotent: the ledger is re-read every pass, so a unit that already has a
grok verdict at a rung is skipped and a killed run can simply be restarted.

    grok_ladder.py --plan          # what it would run, rung by rung
    grok_ladder.py --run           # do it, blocking until the ladder is complete
    grok_ladder.py --report        # grok's curve so far against composer's
"""
from __future__ import annotations

import argparse
import json
import os
import pathlib
import subprocess
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
REPO = pathlib.Path(__file__).resolve().parents[2]
SWEEPS = REPO / "experiments" / "dose_response"
# FLAT cohort names, directly under dose_response. sweep_seq builds its harbor
# --job-name from the cohort argument, so a nested "sweep_grokladder/L3" would put a
# slash in a job name and land the jobs dir somewhere unintended.
COHORT_PREFIX = "sweep_grokladder"
RUNGS = (2, 3, 4, 5, 6)
SOLVER = "grok"
MODEL = os.environ.get("GROK_MODEL", "grok-4.7")


def curves():
    import trial_ledger as TL
    return TL.ledger_by_solver()


def exhausted_by(solver: str, bys) -> list[str]:
    """Units where `solver` has an L6 verdict and never passed any rung.

    Derived, not hardcoded: the roster has to stay correct as verdicts land, and a unit
    that later flips must drop out of it rather than keep drawing trials.
    """
    out = []
    for base, per in bys.items():
        d = per.get(solver) or {}
        if not d.get("6"):
            continue
        passed = any(max(v) > 0 for r, v in d.items()
                     if r.isdigit() and int(r) >= 2 and v)
        if not passed:
            out.append(base)
    return sorted(out)


def have(bys, base: str, rung: int) -> list[float]:
    return ((bys.get(base) or {}).get(SOLVER) or {}).get(str(rung), [])


def staged_dir(base: str, rung: int) -> pathlib.Path | None:
    import glob as _g
    hits = _g.glob(str(SWEEPS / f"sweep_*/{base}-L{rung}"))
    for h in sorted(hits):
        p = pathlib.Path(h)
        if (p / "instruction.md").is_file():
            return p
    return None


def ensure_staged(base: str, rung: int) -> tuple[pathlib.Path | None, str]:
    d = staged_dir(base, rung)
    if d is not None:
        return d, f"already staged at {d.relative_to(REPO)}"
    import escalate
    ok, why = escalate.stage(base, rung, solver=SOLVER)
    if not ok:
        return None, why
    d = staged_dir(base, rung)
    return d, why


def roster(base: str, rung: int) -> None:
    """Record that this cell is wanted, which is how the guard's merged gates are opened.

    The non-flipping rule and the all-solver ceiling are deliberately merged -- a contract
    three failures called broken is evidence for the next model too -- and both defer to
    this roster for a solver holding nothing at the rung. Writing the cell here is the
    explicit statement that we want it measured rather than inferred.
    """
    import trial_guard as TG
    p = REPO / TG.LADDER_BACKFILL_ROSTER
    p.parent.mkdir(parents=True, exist_ok=True)
    line = f"{base} {rung}"
    existing = set()
    if p.is_file():
        existing = {l.strip() for l in p.read_text().splitlines()}
    if line not in existing:
        with open(p, "a") as fh:
            fh.write(line + "\n")


def build_cohort(units: list[tuple[str, pathlib.Path]], rung: int) -> pathlib.Path:
    """One cohort dir per rung, holding only the units that still need that rung.

    sweep_seq walks a cohort and trials everything the guard allows, so pointing it at an
    existing shared cohort (sweep_escalate_L5, say) would run every OTHER unit in there
    too. A purpose-built directory is how the run stays scoped to the roster.
    """
    dest = SWEEPS / f"{COHORT_PREFIX}_L{rung}"
    dest.mkdir(parents=True, exist_ok=True)
    for base, src in units:
        link = dest / f"{base}-L{rung}"
        if link.exists():
            continue
        # cp -al: hardlinks, so a 19MB source tree costs inodes rather than 19MB. The trees
        # are read-only inputs; harbor copies them into the container itself.
        subprocess.run(["cp", "-al", str(src), str(link)], check=True)
    return dest


def launch(cohort: pathlib.Path, rung: int, n: int, log: pathlib.Path) -> int:
    env = dict(os.environ, AGENT="grok-build", MODEL=MODEL,
               GROK_EFFORT=os.environ.get("GROK_EFFORT", "high"),
               GUARD_SOLVER=SOLVER)
    assert cohort.parent == SWEEPS, f"cohort must sit directly under {SWEEPS}"
    cmd = ["bash", str(REPO / "scripts" / "ops" / "sweep_seq.sh"),
           cohort.name, str(rung), str(n)]
    log.parent.mkdir(parents=True, exist_ok=True)
    with open(log, "ab") as fh:
        fh.write(f"\n=== {time.strftime('%H:%M:%S')} rung L{rung}, {n} unit(s)\n".encode())
        fh.flush()
        return subprocess.run(cmd, cwd=REPO, env=env, stdout=fh, stderr=fh).returncode


def plan(bys=None):
    bys = bys or curves()
    units = exhausted_by("composer", bys)
    rows = []
    for rung in RUNGS:
        need = [u for u in units if not have(bys, u, rung)]
        rows.append((rung, need))
    return units, rows


def cmd_plan() -> int:
    units, rows = plan()
    print(f"units composer exhausted at L6 (derived): {', '.join(units) or 'none'}")
    for rung, need in rows:
        mark = "RUN " if need else "done"
        print(f"  L{rung}  {mark}  {', '.join(need) if need else 'grok has every verdict'}")
    total = sum(len(n) for _, n in rows)
    print(f"trials to buy: {total}")
    return 0


def cmd_report() -> int:
    bys = curves()
    for base in exhausted_by("composer", bys) or sorted(
            b for b in bys if (bys[b].get(SOLVER) or {})):
        print(f"\n{base}")
        for who in ("composer", SOLVER):
            d = (bys.get(base) or {}).get(who) or {}
            if not d:
                print(f"  {who:9s} (no verdict)")
                continue
            trail = "  ".join(
                f"L{r}:" + "/".join("PASS" if x > 0 else "fail" for x in d[r])
                for r in sorted(d, key=lambda z: int(z) if z.isdigit() else 99))
            flip = [int(r) for r, v in d.items()
                    if r.isdigit() and int(r) >= 2 and v and max(v) > 0]
            verdict = f"flips at L{min(flip)}" if flip else "EXHAUSTED"
            print(f"  {who:9s} {trail}   -> {verdict}")
    return 0


def cmd_run(max_passes: int) -> int:
    log = REPO / "outputs" / "supervisor" / "grok_ladder.log"
    # max_passes budgets LAUNCHES, not wall-clock iterations. Counting a 120s wait against
    # it means one slow trial -- and these run 20-40 minutes -- burns the whole budget
    # waiting and the loop exits having climbed nothing.
    pass_no = 0
    waits = 0
    while pass_no < max_passes:
        bys = curves()
        units, rows = plan(bys)
        if not units:
            print("no units to climb (none exhausted by composer)")
            return 0
        todo = [(r, n) for r, n in rows if n]
        if not todo:
            print(f"ladder complete for {', '.join(units)}")
            return cmd_report()
        rung, need = todo[0]
        # A sweep already in flight owns its rung. Launching a second one for the same
        # (unit, rung) would run the trial twice -- the per-solver cap counts VERDICTS, and
        # neither run has produced one yet, so nothing would refuse it.
        busy = _running_units()
        clash = sorted(set(need) & busy)
        if clash:
            waits += 1
            if waits % 10 == 1:
                print(f"L{rung}: waiting on in-flight trial(s) for {', '.join(clash)} "
                      f"({waits * 2} min so far)", flush=True)
            if waits > 360:        # 12h; a trial that has not finished is not going to
                print(f"giving up waiting on {', '.join(clash)}", flush=True)
                return cmd_report()
            time.sleep(120)
            continue
        waits = 0
        pass_no += 1
        print(f"[pass {pass_no}] L{rung}: {', '.join(need)}", flush=True)
        staged = []
        for base in need:
            d, why = ensure_staged(base, rung)
            if d is None:
                # A rung that cannot be built is a fact about the unit, recorded and
                # skipped rather than retried forever -- otherwise one unbuildable cell
                # spins this loop until max_passes.
                print(f"  SKIP {base} L{rung}: {why}", flush=True)
                _skip_note(base, rung, why)
                continue
            roster(base, rung)
            staged.append((base, d))
            print(f"  {base}: {why}", flush=True)
        if not staged:
            print(f"  nothing buildable at L{rung}; marking rung unreachable", flush=True)
            continue
        cohort = build_cohort(staged, rung)
        rc = launch(cohort, rung, len(staged), log)
        print(f"  sweep rc={rc}; log {log.relative_to(REPO)}", flush=True)
        after = curves()
        for base, _ in staged:
            v = have(after, base, rung)
            got = "/".join("PASS" if x > 0 else "fail" for x in v) if v else "NO VERDICT"
            print(f"  -> {base} L{rung}: {got}", flush=True)
    print(f"stopped after {max_passes} pass(es)")
    return cmd_report()


def _running_units() -> set[str]:
    """Unit bases with a live container right now, from docker rather than from a pidfile.

    A container name is "<base>-l<rung>__<hash>__env-main-1", lowercased by docker compose,
    so the base is recovered from the prefix. Reading the actual containers means a sweep
    launched by the supervisor, by hand, or by an earlier pass of this loop all count.
    """
    try:
        out = subprocess.run(["docker", "ps", "--format", "{{.Names}}"],
                             capture_output=True, text=True, timeout=60).stdout
    except Exception:
        return set()
    live = set()
    for n in out.split():
        if "__env-main" not in n:
            continue
        stem = n.split("__", 1)[0]
        if "-l" in stem:
            live.add(stem.rsplit("-l", 1)[0])
    return live


def _skip_note(base: str, rung: int, why: str) -> None:
    p = REPO / "outputs" / "supervisor" / "grok_ladder_unbuildable.json"
    try:
        d = json.loads(p.read_text()) if p.is_file() else {}
    except Exception:
        d = {}
    d[f"{base}-L{rung}"] = why
    p.write_text(json.dumps(d, indent=2, sort_keys=True) + "\n")


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--plan", action="store_true")
    ap.add_argument("--run", action="store_true")
    ap.add_argument("--report", action="store_true")
    ap.add_argument("--max-passes", type=int, default=24)
    a = ap.parse_args()
    if a.report:
        return cmd_report()
    if a.run:
        return cmd_run(a.max_passes)
    return cmd_plan()


if __name__ == "__main__":
    raise SystemExit(main())
