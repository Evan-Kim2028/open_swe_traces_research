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


def test_params_go_present_identical_and_unexcised(tmp_path: Path) -> None:
    out = tmp_path / "u"
    F.generate_unit(seed=7, n_funcs=24, ratio=0.8, out=out, proof=False)
    excised = (out / "excised_tree" / "params.go").read_text(encoding="utf-8")
    assert excised == (out / "module" / "params.go").read_text(encoding="utf-8")
    assert 'panic("excised' not in excised
    assert "paramStageMul1" in excised
    # excision and cheat patches must not touch params.go
    patch = (out / "_author" / "excised" / "excision.patch").read_text(encoding="utf-8")
    assert "params.go" not in patch
    assert "params.go" not in (out / "_author" / "cheat.patch").read_text(encoding="utf-8")
    assert "params.go" not in (out / "_author" / "gold.patch").read_text(encoding="utf-8")


def test_repro_runs_in_tree_suite_and_reports_wants(tmp_path: Path) -> None:
    _require_go()
    out = tmp_path / "u"
    F.generate_unit(seed=7, n_funcs=24, ratio=0.8, out=out, proof=True)
    import subprocess

    r = subprocess.run(
        ["bash", "repro.sh"], cwd=out, capture_output=True, text=True, timeout=300, check=False
    )
    assert r.returncode != 0  # excised tree must fail
    blob = r.stdout + r.stderr
    assert "FAIL" in blob
    assert "panic" in blob  # stub panic is the reported symptom
    # worked examples (expected-vs-got) live in the in-tree suite source;
    # they print as soon as the stub panic is repaired but logic is still wrong
    smoke = (out / "excised_tree" / "store_test.go").read_text(encoding="utf-8")
    assert ", want " in smoke
    assert "paramReservedKey" in smoke


def test_coverage_table_maps_every_hidden_test(tmp_path: Path) -> None:
    import re

    out = tmp_path / "u"
    F.generate_unit(seed=7, n_funcs=24, ratio=0.8, out=out, proof=False)
    hidden = "\n".join(
        f.read_text(encoding="utf-8") for f in sorted((out / "hidden").glob("*.go"))
    )
    names = re.findall(r"^func (Test\w+)\s*\(", hidden, re.MULTILINE)
    contract = (out / "_author" / "contract.md").read_text(encoding="utf-8")
    for n in names:
        assert f"| {n} |" in contract, n


def test_fairness_self_audit(tmp_path: Path) -> None:
    _require_go()
    from openswe_traces.synth import fab_fairness

    out = tmp_path / "u"
    F.generate_unit(seed=7, n_funcs=24, ratio=0.8, out=out, proof=True)
    a = fab_fairness.audit_unit(out)
    assert a.ok, {c.name: c.detail for c in a.checks if not c.passed}
    # negative control: excise params.go from the L0 tree -> literals must vanish
    (out / "excised_tree" / "params.go").unlink()
    (out / "excised_tree" / "store_test.go").unlink()  # smoke uses param names
    b = fab_fairness.audit_unit(out)
    assert not b.ok
    assert any(c.name == "hidden_literals" and not c.passed for c in b.checks)
