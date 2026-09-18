from __future__ import annotations

import json
import os
import shutil
import subprocess
from pathlib import Path

import pytest

from openswe_traces.data import ROOT
from openswe_traces.synth.difficulty import (
    design_for_rung,
    find_guard_tests,
    find_perf_gates,
    find_sequence_tests,
    find_sparse_branches,
    is_sequence_test_name,
    name_leakage,
    parse_bench_ns_op,
    pick_contract_drift_site,
    pick_decoy,
    pick_fair_ambiguity,
    pick_feature_excision,
    pick_implicit_invariant,
    pick_sequence_site,
    pick_subsystem_excision,
    pick_two_site_pair,
    race_gate,
    validate_design,
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


def test_openswe_synth_writes_validation_json(tmp_path: Path) -> None:
    _ensure_fixture_indexed()
    out = tmp_path / "task"
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
            "2",
            "--decoys",
            "1",
            "--cross-module",
            "--guard",
            "--out",
            str(out),
        ],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=False,
    )
    assert proc.returncode in {0, 2}, proc.stderr
    val_path = out / "validation.json"
    assert val_path.is_file()
    payload = json.loads(val_path.read_text())
    assert payload["rung"] == 5
    assert payload["cross_module"] is True
    assert payload["guard"] is True
    assert "checks" in payload
    assert "ok" in payload
    design = design_for_rung(FIXTURE, rung=5, hops=1, sites=2, decoys=1, cross_module=True)
    checked = validate_design(
        design, hops=1, sites=2, decoys=1, cross_module=True, guard=True
    )
    assert checked["checks"]["site_found"]
    assert checked["cross_module"] is True


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


def test_pick_sequence_site_and_rung7() -> None:
    _ensure_fixture_indexed()
    assert is_sequence_test_name("TestNextSequence")
    assert is_sequence_test_name("TestLocalOracle")
    assert not is_sequence_test_name("TestNextOnce")
    assert not is_sequence_test_name("TestLocalOracle_UntilExpired")
    site = pick_sequence_site(FIXTURE, n=0)
    assert site is not None
    names = {site.name}
    # Scan a few indices so Next is in the candidate ring.
    for i in range(8):
        s = pick_sequence_site(FIXTURE, n=i)
        if s:
            names.add(s.name)
    assert "Next" in names
    next_site = None
    for i in range(16):
        s = pick_sequence_site(FIXTURE, n=i)
        if s and s.name == "Next":
            next_site = s
            break
    assert next_site is not None
    assert "TestNextSequence" in next_site.sequence_tests
    assert "TestNextOnce" in next_site.single_call_tests
    seq_tests = find_sequence_tests(FIXTURE, "Next")
    assert "TestNextSequence" in seq_tests
    design = design_for_rung(FIXTURE, rung=7, hops=1, sites=1, decoys=0, index=0)
    assert design["rung"] == 7
    assert design.get("sequence")
    leak = name_leakage(
        ["TestNextSequence"],
        "unique=1",
        changed_symbols=["Next"],
        changed_files=["mathx/mathx.go"],
    )
    assert leak["ok"] is True
    leaky = name_leakage(
        ["TestNextSequence"],
        "Next returned 1 twice",
        changed_symbols=["Next"],
        changed_files=["mathx/mathx.go"],
    )
    assert leaky["ok"] is False


def test_pick_feature_excision_and_rung8() -> None:
    _ensure_fixture_indexed()
    exc = pick_feature_excision(FIXTURE, min_functions=3, min_files=2, keep_interface=True)
    assert exc is not None
    assert len(exc.functions) >= 3
    assert len(exc.files) >= 2
    assert exc.tests
    assert exc.keep_interface is True
    names = set(exc.functions)
    found_roundtrip = exc.entry == "RoundTrip"
    for i in range(12):
        e = pick_feature_excision(FIXTURE, min_functions=3, min_files=2, n=i)
        if e:
            names.update(e.functions)
            found_roundtrip = found_roundtrip or e.entry == "RoundTrip"
    assert "RoundTrip" in names or "Pack" in names
    assert found_roundtrip or "Pack" in names
    gone = pick_feature_excision(
        FIXTURE, min_functions=3, min_files=2, keep_interface=False
    )
    assert gone is not None
    assert gone.keep_interface is False
    design = design_for_rung(FIXTURE, rung=8, hops=1, sites=3, decoys=0, index=0)
    assert design["rung"] == 8
    assert design.get("excision")
    assert design["keep_interface"] is True
    design_b = design_for_rung(
        FIXTURE, rung=8, hops=1, sites=6, decoys=0, index=0, keep_interface=False
    )
    assert design_b["keep_interface"] is False
    leak = name_leakage(
        ["TestRoundTrip"],
        "got Point want Point",
        changed_symbols=["Pack"],
        changed_files=["left/left.go"],
    )
    assert leak["ok"] is True
    leaky = name_leakage(
        ["TestRoundTrip"],
        "Pack returned 0",
        changed_symbols=["Pack"],
        changed_files=["left/left.go"],
    )
    assert leaky["ok"] is False


def test_pick_subsystem_excision_perf_gates_and_race() -> None:
    _ensure_fixture_indexed()
    # Fixture is smaller than a real subsystem; the picker still respects mins.
    tiny = pick_subsystem_excision(FIXTURE, min_functions=3, min_files=2, min_lines=10)
    assert tiny is not None
    assert len(tiny.functions) >= 3
    assert len(tiny.files) >= 2
    assert tiny.keep_interface is False
    none = pick_subsystem_excision(FIXTURE, min_functions=500, min_files=50, min_lines=10_000)
    assert none is None
    design = design_for_rung(
        FIXTURE, rung=8, hops=1, sites=12, decoys=0, index=0, keep_interface=False
    )
    assert design["rung"] == 8
    assert design.get("subsystem") is True
    gates = find_perf_gates(FIXTURE)
    names = {g.name for g in gates}
    assert "BenchmarkAdd" in names
    assert any(g.kind == "benchmark" for g in gates)
    parsed = parse_bench_ns_op(
        "BenchmarkAdd-8\t12345\t67.0 ns/op\nPASS\nok\tfixturehost/mathx\t0.2s\n",
        "BenchmarkAdd",
    )
    assert parsed == 67.0
    site = race_gate(FIXTURE, n=0)
    assert site is not None
    assert site.file_path.endswith("mathx.go")
    assert "mutex" in site.reason.lower() or "map" in site.reason.lower()


def test_pick_sized_excision_band() -> None:
    from openswe_traces.synth.difficulty import pick_sized_excision

    _ensure_fixture_indexed()
    tiny = pick_sized_excision(
        FIXTURE, min_functions=3, max_functions=20, min_lines=10, max_lines=500
    )
    assert tiny is not None
    assert 3 <= len(tiny.functions) <= 20
    none = pick_sized_excision(
        FIXTURE, min_functions=3, max_functions=8, min_lines=10_000, max_lines=20_000
    )
    assert none is None

