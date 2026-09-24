#!/usr/bin/env python3
"""Stage regenerated goa L2 contracts into experiments/dose_response/sweep_goa_rc/.

For each goa unit: copy the existing L2 task tree (sweep_goa_L2 for the 12
trialed units, which carries environment/src; batch2 <u>-L2 + grafted src for
the 3 untrialed), then splice the reconcile-v2 contract over instruction.md,
keeping the bug-report/reproduce/no-web tail. Dest dirs are <unit>-L2rc.
"""
import shutil
import sys
from pathlib import Path

sys.path.insert(0, "/home/evan/Documents/oswt-RCFIX/src")
from openswe_traces.synth.reconcile import splice_contract

UNITS = [
    "dupexpr", "exprhash", "httpclienterr", "httpencoding", "httperrresp",
    "httpmux", "importalias", "mappedattr", "namescope", "reqidgen",
    "retrypolicy", "sampler", "skipwriter", "svcerror", "traceopts",
]
TRIALED = [u.name[:-3] for u in Path(
    "/home/evan/Documents/open_swe_traces_research/experiments/dose_response/sweep_goa_L2"
).iterdir()]

MAIN = Path("/home/evan/Documents/open_swe_traces_research")
SWEEP_L2 = MAIN / "experiments/dose_response/sweep_goa_L2"
SWEEP_L0 = MAIN / "experiments/dose_response/sweep_goa"
BATCH2 = Path("/home/evan/Documents/oswt-VFgoa/experiments/pipeline/tasks_batch2/goa")
CONTRACTS = Path("/home/evan/Documents/oswt-RECONCILE/outputs/reconcile/goa")
DEST = MAIN / "experiments/dose_response/sweep_goa_rc"


def main() -> int:
    DEST.mkdir(parents=True, exist_ok=True)
    for unit in UNITS:
        dest = DEST / f"{unit}-L2rc"
        if dest.exists():
            shutil.rmtree(dest)
        if unit in TRIALED:
            shutil.copytree(SWEEP_L2 / f"{unit}-L2", dest, symlinks=True)
        else:
            shutil.copytree(BATCH2 / f"{unit}-L2", dest, symlinks=True)
            if not (dest / "environment/src").exists():
                src = SWEEP_L0 / f"{unit}-L0/environment/src"
                shutil.copytree(src, dest / "environment/src", symlinks=True)
        contract = (CONTRACTS / unit / "contract.md").read_text()
        instr = dest / "instruction.md"
        instr.write_text(splice_contract(instr.read_text(), contract))
        n_rows = contract.count("\n| `Test")
        print(f"{unit}: rows={n_rows} dest={dest.name} src={'trialed' if unit in TRIALED else 'batch2+L0src'}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
