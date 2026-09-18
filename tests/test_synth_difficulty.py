from __future__ import annotations

import shutil
import subprocess
from pathlib import Path

import pytest

from openswe_traces.data import ROOT
from openswe_traces.synth.difficulty import (
    find_guard_tests,
    find_sparse_branches,
    no_import_edge,
    package_imports,
    pick_contract_drift_site,
    pick_decoy,
    pick_three_site,
    pick_two_site_pair,
    select_construction,
)

FIXTURE = ROOT / "experiments" / "codegraph_bugs" / "fixture_host"


def _ensure_fixture_indexed() -> None:
    if not shutil.which("codegraph"):
        pytest.skip("codegraph CLI not on PATH")
    proc = subprocess.run(
        ["codegraph", "init", "-y", str(FIXTURE)],
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        pytest.skip(f"codegraph init failed: {proc.stderr}")
    proc = subprocess.run(
        ["codegraph", "index", str(FIXTURE)],
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        pytest.skip(f"codegraph index failed: {proc.stderr}")


def test_pick_encode_decode_and_three_site() -> None:
    _ensure_fixture_indexed()
    pair = pick_two_site_pair(FIXTURE, require_no_import=False, cross_package=False)
    assert pair is not None
    names = {pair.a.name, pair.b.name}
    assert "Encode" in names
    assert "Decode" in names
    triple = pick_three_site(FIXTURE)
    assert triple is not None
    assert {s.name for s in triple.sites} == {"Compose", "Extract", "FromTime"}


def test_pick_cross_package_no_import() -> None:
    _ensure_fixture_indexed()
    imports = package_imports(FIXTURE)
    assert no_import_edge(imports, "left", "right")
    pair = pick_two_site_pair(FIXTURE, require_no_import=True, cross_package=True)
    assert pair is not None
    pkgs = {pair.a.package, pair.b.package}
    assert "left" in pkgs
    assert "right" in pkgs
    assert pair.no_import_edge


def test_pick_decoy_and_guard_on_add_path() -> None:
    _ensure_fixture_indexed()
    decoy = pick_decoy(FIXTURE, ["Add", "SumClamped", "TestSumClamped"], true_cause="Add")
    assert decoy is not None
    assert decoy.name == "SumClamped"
    guards = find_guard_tests(FIXTURE, "Add", exclude=["TestSumClamped"])
    assert "TestAdd" in guards


def test_find_sparse_branches_coverprofile(tmp_path: Path) -> None:
    profile = tmp_path / "c.out"
    profile.write_text(
        "mode: set\n"
        "fixturehost/mathx/mathx.go:4.24,6.2 1 1\n"
        "fixturehost/mathx/mathx.go:10.32,12.13 1 0\n"
    )
    sparse = find_sparse_branches(profile, max_count=0)
    assert any(s["count"] == 0 for s in sparse)
    func = tmp_path / "func.out"
    func.write_text("fixturehost/mathx/mathx.go:9:\tClamp\t20.0%\ntotal:\t\t(statements)\t80.0%\n")
    sparse_fn = find_sparse_branches(func)
    assert any(s.get("func") == "Clamp" for s in sparse_fn)


def test_select_construction_rung3() -> None:
    _ensure_fixture_indexed()
    three = select_construction(FIXTURE, rung=3, hops=1, sites=3, decoys=0)
    assert three.triple is not None
    assert len(three.picked_sites) == 3
    two = select_construction(FIXTURE, rung=3, hops=1, sites=2, decoys=1)
    assert two.pair is not None
    assert two.pair.no_import_edge
    assert two.decoy_list or two.picked_sites


def test_pick_contract_drift_min_hops() -> None:
    _ensure_fixture_indexed()
    site = pick_contract_drift_site(FIXTURE, min_hops=1)
    assert site is not None
    assert site.hops is not None and site.hops >= 1
    deep = pick_contract_drift_site(FIXTURE, min_hops=99)
    assert deep is None
