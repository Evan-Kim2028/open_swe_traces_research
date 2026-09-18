from __future__ import annotations

import os
import shutil
import subprocess
from pathlib import Path

import pytest

from openswe_traces.data import ROOT
from openswe_traces.synth.difficulty import (
    design_for_rung,
    find_guard_tests,
    find_sparse_branches,
    name_leakage,
    pick_contract_drift_site,
    pick_decoy,
    pick_fair_ambiguity,
    pick_implicit_invariant,
    pick_two_site_pair,
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
    subprocess.run(
        ["codegraph", "index", str(FIXTURE)],
        capture_output=True,
        text=True,
        check=False,
    )


def test_pick_contract_drift_site_min_hops() -> None:
    _ensure_fixture_indexed()
    site = pick_contract_drift_site(FIXTURE, min_hops=1, n=0)
    assert site is not None
    assert site.hops >= 1
    assert site.name
    assert site.path
    deep = pick_contract_drift_site(FIXTURE, min_hops=2, n=0)
    if deep is not None:
        assert deep.hops >= 2
        assert deep.name
        assert deep.path


def test_pick_decoy_intermediate_and_sibling() -> None:
    _ensure_fixture_indexed()
    # Add → SumClamped → TestSumClamped: SumClamped is the path decoy.
    decoy = pick_decoy(FIXTURE, ["Add", "SumClamped", "TestSumClamped"], n=0)
    assert decoy is not None
    assert decoy.name == "SumClamped"
    # Encode/Decode are inverse siblings of TestEncodeDecode.
    sib = pick_decoy(FIXTURE, ["Encode", "TestEncodeDecode"], n=0)
    assert sib is not None
    assert sib.name == "Decode"
    assert "TestEncodeDecode" in sib.guard_tests


def test_pick_two_site_pair_and_guard_tests() -> None:
    _ensure_fixture_indexed()
    pair = pick_two_site_pair(FIXTURE, n=0)
    assert pair is not None
    assert pair.a_file != pair.b_file
    assert pair.a == pair.b
    assert not pair.same_package
    assert pair.import_edge is False
    guards = find_guard_tests(FIXTURE, "Add", max_hops=3)
    assert "TestAdd" in guards or "TestSumClamped" in guards


def test_find_sparse_branches_func_and_profile(tmp_path: Path) -> None:
    func = tmp_path / "cover.func"
    func.write_text(
        "fixturehost/mathx/mathx.go:4:\tAdd\t100.0%\n"
        "fixturehost/mathx/mathx.go:9:\tClamp\t40.0%\n"
        "fixturehost/codec/codec.go:4:\tEncode\t0.0%\n"
        "total:\t\t(statements)\t50.0%\n"
    )
    sparse = find_sparse_branches(func, max_pct=50.0)
    names = {(fp, fn) for fp, fn, _pct in sparse}
    assert ("mathx.go", "Clamp") in names or any(fn == "Clamp" for _fp, fn, _ in sparse)
    assert any(fn == "Encode" for _fp, fn, _ in sparse)
    assert not any(fn == "Add" for _fp, fn, _pct in sparse)

    profile = tmp_path / "cover.out"
    profile.write_text(
        "mode: set\n"
        "fixturehost/mathx/mathx.go:4.1,6.2 2 1\n"
        "fixturehost/codec/codec.go:4.1,6.2 2 0\n"
    )
    files = find_sparse_branches(profile, max_pct=50.0)
    assert any("codec.go" in fp for fp, _fn, _pct in files)


def test_design_for_rung_jsonable() -> None:
    _ensure_fixture_indexed()
    design = design_for_rung(FIXTURE, rung=5, hops=1, sites=1, decoys=1, index=0)
    assert design["rung"] == 5
    assert "site" in design
    assert "decoys" in design
    if design["site"]:
        assert design["site"]["hops"] >= 1


def test_openswe_synth_cli_smoke() -> None:
    _ensure_fixture_indexed()
    env = os.environ.copy()
    proc = subprocess.run(
        [
            "uv",
            "run",
            "python",
            "-m",
            "openswe_traces.synth.difficulty",
            "--repo",
            str(FIXTURE),
            "--rung",
            "5",
            "--hops",
            "1",
            "--sites",
            "1",
            "--decoys",
            "1",
        ],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=False,
        env=env,
    )
    assert proc.returncode == 0, proc.stderr
    assert '"rung": 5' in proc.stdout


def test_pick_fair_ambiguity_inverse_and_name_leakage() -> None:
    _ensure_fixture_indexed()
    amb = pick_fair_ambiguity(FIXTURE, n=0)
    assert amb is not None
    assert amb.cause == "Encode"
    assert amb.decoy.name == "Decode"
    assert "TestEncodeDecode" in amb.decoy.guard_tests
    inv = pick_implicit_invariant(FIXTURE, n=0)
    assert inv is not None
    assert inv.name == "Clamp"
    leak = name_leakage(
        ["TestSumClamped"],
        "expected 4 got 9",
        changed_symbols=["Add"],
        changed_files=["mathx/mathx.go"],
    )
    assert leak["ok"] is True
    leaky = name_leakage(
        ["TestParseKeyspaceID"],
        "ParseKeyspaceID returned 0",
        changed_symbols=["ParseKeyspaceID"],
        changed_files=["internal/apicodec/codec.go"],
    )
    assert leaky["ok"] is False
    assert "ParseKeyspaceID" in leaky["leaked_symbols"]


def test_design_for_rung_fair_ambiguity_payload() -> None:
    _ensure_fixture_indexed()
    design = design_for_rung(FIXTURE, rung=5, hops=1, sites=1, decoys=1, index=0)
    assert design.get("fair_ambiguity")
    assert design["fair_ambiguity"]["cause"] == "Encode"
    assert any(d["name"] == "Decode" for d in design["decoys"])
