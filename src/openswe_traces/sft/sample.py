"""Sample N traces from one harness/teacher into a compact JSONL for Kaggle SFT.

Single source (mini-swe-agent + Qwen3.6-27B, non-thinking, one `bash` tool) so the
smoke run measures throughput, not source mix. Tool observations truncated to keep
sequences bounded; they are loss-masked anyway.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path

import duckdb

from ..data import DATA_ROOT, ROOT

SRC_GLOB = str(DATA_ROOT / "minisweagent" / "qwen36_27b" / "swe-rebench-v2" / "*.parquet")
OUT_DIR = ROOT / "experiments" / "kaggle_smoke" / "data"


def build_sample(
    n: int = 1000,
    *,
    tool_max_chars: int = 1500,
    seed: int = 0,
    src: str = SRC_GLOB,
    out_dir: Path = OUT_DIR,
) -> Path:
    con = duckdb.connect()
    rows = con.sql(f"""
      select trajectory_id, instance_id, language, resolved, messages, tools
      from '{src}'
      using sample reservoir({n} rows) repeatable ({seed})
    """).fetchall()
    con.close()

    out_dir.mkdir(parents=True, exist_ok=True)
    path = out_dir / f"sample_{n}.jsonl"
    n_msgs = 0
    with path.open("w") as f:
        for tid, iid, lang, res, msgs, tools in rows:
            clean = []
            for m in msgs:
                role = m["role"]
                content = m["content"] or ""
                if role == "tool" and len(content) > tool_max_chars:
                    content = content[:tool_max_chars] + "\n...[truncated]"
                d = {"role": role, "content": content}
                if role == "assistant" and m.get("tool_calls"):
                    tcs = []
                    for tc in m["tool_calls"]:
                        fn = dict(tc["function"])
                        try:
                            fn["arguments"] = json.loads(fn["arguments"])
                        except Exception:  # noqa: BLE001, S110 (leave raw value when not JSON)
                            pass
                        tcs.append({"id": tc["id"], "type": tc["type"], "function": fn})
                    d["tool_calls"] = tcs
                clean.append(d)
            n_msgs += len(clean)
            f.write(
                json.dumps(
                    {
                        "trajectory_id": tid,
                        "instance_id": iid,
                        "language": lang,
                        "resolved": res,
                        "messages": clean,
                        "tools": [json.loads(t) for t in tools],
                    }
                )
                + "\n"
            )
    print(
        f"wrote {path} rows={len(rows)} avg_msgs={n_msgs / len(rows):.1f} "
        f"bytes={path.stat().st_size / 1e6:.1f}MB"
    )
    return path


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--n", type=int, default=1000)
    ap.add_argument("--tool-max-chars", type=int, default=1500)
    ap.add_argument("--seed", type=int, default=0)
    a = ap.parse_args()
    build_sample(a.n, tool_max_chars=a.tool_max_chars, seed=a.seed)
