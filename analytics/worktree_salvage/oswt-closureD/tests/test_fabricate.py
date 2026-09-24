"""Tests for the fabricated-repo generator (src/openswe_traces/synth/fabricate.py)."""

from __future__ import annotations

import json
import shutil
from pathlib import Path

import pytest

from openswe_traces.synth import fabricate as F


def _require_go() -> None:
    if shutil.which("go") is None:
        pytest.skip("go toolchain not on PATH")


def test_store_unit_small_compiles_and_proves(tmp_path: Path) -> None:
    _require_go()
    out = tmp_path / "store-u"
    r = F.generate_unit(seed=3, n_funcs=10, ratio=0.8, out=out, proof=True)
    assert r.stats.n_funcs == 10
    assert abs(r.stats.ratio - 0.8) <= 0.15
    assert r.proof["ok"], {k: v for k, v in r.proof.items() if k.endswith("_log")}
    author = out / "_author"
    for name in ("api.md", "bugreport.md", "contract.md", "closure.md", "difficulty.md", "gold.patch", "cheat.patch"):
        assert (author / name).is_file(), name
    assert (author / "excised" / "excision.patch").is_file()
    assert (out / "module" / "go.mod").is_file()
    assert (out / "module" / "store.go").is_file()
    assert (out / "excised_tree" / "store.go").is_file()
    assert (out / "hidden" / "store_bb_test.go").is_file()
    assert (out / "repro.sh").is_file()
    manifest = json.loads((out / "manifest.json").read_text(encoding="utf-8"))
    assert manifest["ratio_achieved"] == pytest.approx(r.stats.ratio)


def test_sched_unit_small_compiles_and_proves(tmp_path: Path) -> None:
    _require_go()
    out = tmp_path / "sched-u"
    r = F.generate_unit(seed=5, n_funcs=15, ratio=0.8, out=out, domain="sched", proof=True)
    assert r.stats.n_funcs == 15
    assert abs(r.stats.ratio - 0.8) <= 0.2
    assert r.proof["ok"], {k: v for k, v in r.proof.items() if k.endswith("_log")}


def test_generation_is_deterministic(tmp_path: Path) -> None:
    a = tmp_path / "a"
    b = tmp_path / "b"
    F.generate_unit(seed=7, n_funcs=24, ratio=0.8, out=a, proof=False)
    F.generate_unit(seed=7, n_funcs=24, ratio=0.8, out=b, proof=False)
    for rel in ("module/store.go", "module/store_test.go", "hidden/store_bb_test.go", "_author/gold.patch", "_author/cheat.patch"):
        assert (a / rel).read_text(encoding="utf-8") == (b / rel).read_text(encoding="utf-8"), rel


def test_bugreport_has_no_symbol_names(tmp_path: Path) -> None:
    F.generate_unit(seed=7, n_funcs=24, ratio=0.8, out=tmp_path / "u", proof=False)
    report = (tmp_path / "u" / "_author" / "bugreport.md").read_text(encoding="utf-8")
    for leaked in ("Put", "Get", "BucketOf", "NewStore", "mixPipe", "canon", "store.go", "store_test.go", "Store"):
        assert leaked not in report, leaked
    assert "./repro.sh" in report
    assert "Do NOT use web search" in report


def test_measured_ratio_matches_manifest_and_stats(tmp_path: Path) -> None:
    _require_go()
    out = tmp_path / "u"
    r = F.generate_unit(seed=7, n_funcs=24, ratio=2.0, out=out, proof=True)
    manifest = json.loads((out / "manifest.json").read_text(encoding="utf-8"))
    assert manifest["ratio_achieved"] == pytest.approx(r.stats.ratio)
    assert manifest["edges"]["internal"] == r.stats.internal
    assert manifest["edges"]["boundary_in"] == r.stats.in_edges
    assert manifest["edges"]["boundary_out"] == r.stats.out_edges
    assert len(manifest["excision"]["S"]) == len(r.stats.S)
