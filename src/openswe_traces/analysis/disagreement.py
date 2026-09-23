"""What predicts Composer and Devin disagreeing about how hard a task is.

    uv run python -m openswe_traces.analysis.disagreement

A task's grade for one model is the first ladder step it passes (L1, L2, L3-4, L5-6) or
"none" when it fails through the test file. On tasks both models graded, the gap is Devin's
step minus Composer's. This joins that gap against features recorded at staging
(outputs/unit_features.jsonl): answer-key size and files touched from the gold patch, hidden
test functions and assertions, and the length of the bug report and the full description.

Two questions per feature. Does it differ between tasks where the models agree and tasks
where they disagree (Mann-Whitney U)? Does it track the signed gap, or each model's own grade
(Spearman)? With about 76 tasks, treat a single p-value near 0.05 as a lead, not a finding.
"""
import collections
import json
import sys

from scipy import stats

from openswe_traces.ladder import ledger as TL

FEATURES = "outputs/unit_features.jsonl"
STEP = {"0": 0, "2": 1, "3": 2, "4": 2, "5": 3, "6": 3}
NONE = 4
NAMES = {
    "gold_add": "answer-key lines added",
    "gold_files": "files the answer key touches",
    "testfns": "hidden test functions",
    "asserts": "hidden assertions",
    "bug_words": "bug report words (L1)",
    "contract_words": "full description words (L2)",
}


def grade(rungs):
    passed = sorted(STEP[r] for r, v in rungs.items() if r in STEP and v and max(v) > 0)
    if passed:
        return passed[0]
    return NONE if rungs.get("5") or rungs.get("6") else None


def features(path=FEATURES):
    """base -> feature dict, the latest record per (base, rung) merged across rungs."""
    latest = {}
    for line in open(path):
        try:
            r = json.loads(line)
        except ValueError:
            continue
        key = (r.get("base"), r.get("rung"))
        if key[0] and (key not in latest or r.get("t", 0) >= latest[key].get("t", 0)):
            latest[key] = r
    out = collections.defaultdict(dict)
    for (base, rung), r in latest.items():
        f = out[base]
        for k in ("gold_add", "gold_files", "testfns", "asserts"):
            if r.get(k) is not None:
                f.setdefault(k, r[k])
        if rung == "0" and r.get("instr_words"):
            f["bug_words"] = r["instr_words"]
        if rung == "2" and r.get("instr_words"):
            f["contract_words"] = r["instr_words"]
    return out


def rows(jobs_dir=TL.JOBS):
    feats = features()
    out = []
    for base, d in TL.ledger_by_solver(jobs_dir).items():
        if "composer" not in d or "devin" not in d:
            continue
        c, v = grade(d["composer"]), grade(d["devin"])
        if c is None or v is None:
            continue
        out.append({"base": base, "composer": c, "devin": v, "gap": v - c, **feats.get(base, {})})
    return out


def analyse(rs):
    res = {"tasks": len(rs), "agree": sum(r["gap"] == 0 for r in rs), "features": {}}
    for k, name in NAMES.items():
        have = [r for r in rs if r.get(k) is not None]
        if len(have) < 10:
            continue
        same = [r[k] for r in have if r["gap"] == 0]
        diff = [r[k] for r in have if r["gap"] != 0]
        u = stats.mannwhitneyu(same, diff) if same and diff else None
        med = lambda xs: sorted(xs)[len(xs) // 2] if xs else None
        res["features"][name] = {
            "n": len(have),
            "median when agree": med(same),
            "median when disagree": med(diff),
            "agree vs disagree p": round(u.pvalue, 3) if u else None,
            "rho with |gap|": round(stats.spearmanr([r[k] for r in have],
                                                    [abs(r["gap"]) for r in have])[0], 2),
            "rho with signed gap": round(stats.spearmanr([r[k] for r in have],
                                                         [r["gap"] for r in have])[0], 2),
            "rho with composer grade": round(stats.spearmanr([r[k] for r in have],
                                                             [r["composer"] for r in have])[0], 2),
            "rho with devin grade": round(stats.spearmanr([r[k] for r in have],
                                                          [r["devin"] for r in have])[0], 2),
        }
    return res


def cli():
    res = analyse(rows())
    if "--json" in sys.argv:
        print(json.dumps(res, indent=1))
        return
    print(f"{res['tasks']} tasks graded by both, {res['agree']} at the same step")
    for name, f in res["features"].items():
        print(f"\n{name}  (n={f['n']})")
        for k, v in f.items():
            if k != "n":
                print(f"  {k:26s} {v}")


if __name__ == "__main__":
    cli()
