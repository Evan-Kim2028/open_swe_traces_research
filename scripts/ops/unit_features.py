#!/usr/bin/env python3
"""Snapshot pre-trial features of a staged unit, so a too-easy predictor can be calibrated.

We have 74 known-easy units but only 27 still have their hidden tests on disk — the staged
copies were cleaned up after trialling, taking the evidence with them. Medians separate easy
from hard well (28 vs 41 assertions, 6 vs 8 test functions, ~80% separation) but 27 samples
is far too thin to set a production threshold.

This records features at STAGING time, when they still exist, keyed by unit. Join against
the verdict ledger later and the threshold calibrates itself on hundreds of units.

  unit_features.py <stage_dir>     append features for every unit staged there
  unit_features.py --calibrate     join with verdicts and print the threshold table
"""
import sys, os, re, json, glob, time

OUT = "outputs/unit_features.jsonl"


def features(unit_dir):
    f = {}
    gp = os.path.join(unit_dir, "patches", "gold.patch")
    if os.path.exists(gp):
        t = open(gp, errors="replace").read()
        f["gold_add"] = sum(1 for l in t.split("\n") if l.startswith("+") and not l.startswith("+++"))
        f["gold_files"] = t.count("diff --git")
    tst = (glob.glob(os.path.join(unit_dir, "tests", "hidden", "*_test.go"))
           + glob.glob(os.path.join(unit_dir, "tests", "*_test.go")))
    if tst:
        tt = "".join(open(x, errors="replace").read() for x in tst)
        f["testfns"] = len(re.findall(r"^func Test", tt, re.M))
        f["asserts"] = len(re.findall(r"\bt\.(Errorf|Fatalf|Error|Fatal)\b", tt))
        f["subtests"] = len(re.findall(r"\bt\.Run\(", tt))
        f["testlines"] = tt.count("\n")
    ins = os.path.join(unit_dir, "instruction.md")
    if os.path.exists(ins):
        f["instr_words"] = len(open(ins, errors="replace").read().split())
    return f


def snapshot(stage):
    n = 0
    with open(OUT, "a") as fh:
        for d in sorted(glob.glob(os.path.join(stage, "*/"))):
            unit = os.path.basename(d.rstrip("/"))
            f = features(d)
            if not f:
                continue
            f.update(unit=unit, base=unit.rsplit("-L", 1)[0],
                     rung=unit.rsplit("-L", 1)[-1][:1] if "-L" in unit else "?",
                     t=int(time.time()), stage=os.path.basename(stage.rstrip("/")))
            fh.write(json.dumps(f) + "\n")
            n += 1
    print(f"recorded {n} unit(s) from {stage} -> {OUT}")


def calibrate():
    sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
    import trial_ledger
    per = trial_ledger.ledger()
    easy = {u for u, d in per.items() if d.get("0") and max(d["0"]) > 0}
    hard = {u for u, d in per.items() if d.get("0") and max(d["0"]) == 0}
    rows = {}
    if os.path.exists(OUT):
        for ln in open(OUT):
            try:
                r = json.loads(ln)
            except Exception:
                continue
            if r.get("rung") == "0":
                rows.setdefault(r["base"], r)
    E = [r for b, r in rows.items() if b in easy]
    H = [r for b, r in rows.items() if b in hard]
    print(f"calibration set: {len(E)} easy / {len(H)} hard")
    if len(E) < 40:
        print("NOT ENOUGH DATA. Need >=40 easy units before trusting a threshold;\n"
              "at 27 the best precision was ~70% with an interval too wide to use.")
    for key in ("asserts", "testfns"):
        vals = sorted({r[key] for r in E + H if key in r})
        if not vals:
            continue
        print(f"\n  {key} <= T     easy caught  recall  hard lost  precision")
        for t in vals[::max(1, len(vals) // 8)]:
            ec = sum(1 for r in E if r.get(key, 1e9) <= t)
            hc = sum(1 for r in H if r.get(key, 1e9) <= t)
            p = ec / (ec + hc) if ec + hc else 0
            print(f"    {t:<10d} {ec:10d} {ec/max(1,len(E)):7.0%} {hc:10d} {p:10.0%}")


if __name__ == "__main__":
    if "--calibrate" in sys.argv:
        calibrate()
    elif len(sys.argv) > 1:
        snapshot(sys.argv[1])
    else:
        print(__doc__)
