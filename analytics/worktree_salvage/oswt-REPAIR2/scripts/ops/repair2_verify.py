#!/usr/bin/env python3
"""REPAIR2 verification sweep: reaudit + lint + preflight per staged unit.

  repair2_verify.py <staged unit dir> [<staged unit dir> ...]

For each unit: fresh contract-gap read (cache-keyed on repaired files),
task_lint literal budget, and docker preflight with a freshly generated
cheat.patch. Appends one results row per unit to outputs/repair2/results.jsonl.
Resume-safe: units already in results.jsonl are skipped unless --force.
"""
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "src"))

from openswe_traces.gate import repair2  # noqa: E402
import task_lint  # noqa: E402

SWEEP = Path("experiments/dose_response")
SRCS = {
    "client-go-memdbstaging-L2": "sweep_tail_L2",
    "client-go-onregionerror-L2": "sweep_tail_L2",
    "client-go-replicaselector-L2": "sweep_tail_L2",
    "mvccread-L2": "sweep_clientgo_L2",
    "commitobj-L2": "sweep_gogit_L2",
    "fsrefs-L2": "sweep_gogit_L2",
    "indexdec-L2": "sweep_gogit_L2",
    "merklediff-L2": "sweep_gogit_L2",
    "packscan-L2": "sweep_gogit_L2",
    "refnames-L2": "sweep_gogit_L2",
    "refspec-L2": "sweep_gogit_L2",
    "revparse-L2": "sweep_gogit_L2",
    "treeobj-L2": "sweep_gogit_L2",
    "ulreq-L2": "sweep_gogit_L2",
    "confparse-L2": "sweep_nats_L2",
    "cronparse-L2": "sweep_nats_L2",
    "ldapdn-L2": "sweep_nats_L2",
    "gin-enginecfg-L2": "sweep_gin2_L2",
    "gin-negotiate-L2": "sweep_gin2_L2",
    "gin-mailfmt-L2": "sweep_xrepo20",
    "helm-chartdl-L2": "sweep_helm2_L2",
    "helm-chartrepo-L2": "sweep_helm2_L2",
    "helm-depresolver-L2": "sweep_L2",
    "helm-httpgetter-L2": "sweep_helm2_L2",
    "kops-clustervalid-L2": "sweep_L2",
    "kops-difftext-L2": "sweep_kops2_L2",
    "kops-taintparse-L2": "sweep_kops2_L2",
}


def gaps_before(unit: str) -> dict:
    per = repair2.load_gaps()
    kinds = {"ARBITRARY": 0, "DERIVABLE": 0, "COUNTER": 0}
    miss = conf = 0
    for g in per.get(unit, []):
        if g["text"].strip() == "NONE" or g["kind"] == "UNKNOWN":
            continue
        if g["gtype"] == "missing":
            miss += 1
        else:
            conf += 1
        if g["kind"] in kinds:
            kinds[g["kind"]] += 1
    return {"missing": miss, "conflicts": conf, "kinds": kinds}


def verify_one(unit: Path) -> dict:
    name = unit.name
    row = {"unit": name, "src": f"{SRCS.get(name, '?')}/{name}"}
    row["gaps_before"] = gaps_before(name)
    src = SWEEP / SRCS[name] / name
    row["literals_before"] = list(repair2.literal_budget(src)) if src.is_dir() else None
    row["literals_after"] = list(repair2.literal_budget(unit))

    findings = task_lint.lint(unit)
    row["lint"] = {"blocks": [m for s, c, m in findings if s == "BLOCK"],
                   "warns": len([f for f in findings if f[0] == "WARN"])}

    r = repair2.reaudit(unit)
    row["gaps_after"] = {"missing": r["missing"], "conflicts": r["conflicts"],
                         "kinds": r["kinds"]}
    row["reaudit_answer"] = r["answer"]

    pf = repair2.preflight(unit, repair2.load_api_key())
    row["preflight"] = {"bare": pf.bare_reward, "gold": pf.gold_reward,
                        "cheat": pf.cheat_reward, "verdict": pf.verdict,
                        "note": pf.note}
    if pf.cheat_patch:
        (unit / "tests" / "cheat.patch").write_text(pf.cheat_patch)
        p2 = unit / "patches" / "cheat.patch"
        if p2.parent.is_dir():
            p2.write_text(pf.cheat_patch)

    row["fair_at_rung"] = ("L0" if row["gaps_after"]["kinds"]["ARBITRARY"] == 0
                           else "none (unfair until assertion changes)")
    return row


def main(argv: list[str]) -> int:
    force = "--force" in argv
    units = [Path(a) for a in argv if not a.startswith("--")]
    done = set()
    if repair2.RESULTS_PATH.is_file():
        for l in repair2.RESULTS_PATH.read_text().splitlines():
            try:
                done.add(json.loads(l)["unit"])
            except Exception:
                pass
    for u in units:
        if u.name in done and not force:
            print(f"=== {u.name}: already verified, skip", flush=True)
            continue
        print(f"=== {u.name} ===", flush=True)
        try:
            row = verify_one(u)
        except Exception as e:
            print(f"  verify failed: {e}", flush=True)
            row = {"unit": u.name, "error": str(e)}
        with repair2.RESULTS_PATH.open("a") as fh:
            fh.write(json.dumps(row) + "\n")
        print(json.dumps({k: v for k, v in row.items()
                          if k != "reaudit_answer"}, indent=1)[:2000], flush=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
