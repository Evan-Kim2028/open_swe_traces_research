"""Export stratified validation sample and write manual rung labels."""

from __future__ import annotations

import argparse
import json

import pandas as pd

from openswe_traces.data import ROOT
from openswe_traces.rung_analysis import (
    VALIDATION_JSON,
    agreement_stats,
    confusion_matrix,
    load_rung_frame,
    stratified_validation_sample,
)

SAMPLE_JSON = ROOT / "outputs" / "rung_validation_sample.json"


def export_sample(n: int = 200, seed: int = 42) -> pd.DataFrame:
    rung_df = load_rung_frame()
    sample = stratified_validation_sample(rung_df, n=n, seed=seed)
    records = []
    for _, row in sample.iterrows():
        records.append(
            {
                "instance_id": row["instance_id"],
                "rung_heuristic": int(row["rung"]),
                "task_text": row["task_text"],
                "word_count": int(row["word_count"]),
                "has_repro": bool(row["has_repro"]),
                "has_expected_actual": bool(row["has_expected_actual"]),
                "has_test_names": bool(row["has_test_names"]),
                "has_signature": bool(row["has_signature"]),
                "has_test_code": bool(row["has_test_code"]),
            }
        )
    SAMPLE_JSON.parent.mkdir(parents=True, exist_ok=True)
    SAMPLE_JSON.write_text(json.dumps(records, indent=2))
    print(f"Wrote {len(records)} samples → {SAMPLE_JSON.relative_to(ROOT)}")
    return sample


def report_manual() -> None:
    manual = pd.read_json(VALIDATION_JSON)
    stats = agreement_stats(manual)
    print(f"Exact: {100 * stats['exact']:.1f}%  within±1: {100 * stats['within_1']:.1f}%  ρ={stats['spearman']:.3f}")
    print(confusion_matrix(manual).to_string())


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--export", action="store_true", help="Export stratified sample JSON")
    parser.add_argument("--report", action="store_true", help="Print agreement from manual JSON")
    parser.add_argument("-n", type=int, default=200)
    args = parser.parse_args()
    if args.export:
        export_sample(n=args.n)
    elif args.report:
        report_manual()
    else:
        parser.print_help()


if __name__ == "__main__":
    main()
