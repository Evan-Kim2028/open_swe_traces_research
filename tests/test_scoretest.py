from __future__ import annotations

import shutil
import subprocess
from pathlib import Path

import pytest

from openswe_traces.data import ROOT
from openswe_traces.synth.difficulty import FeatureExcision, collect_feature_excisions
from openswe_traces.synth.scoretest import (
    RANK_COMPONENT,
    is_used,
    obfuscate_relpath,
    pick_extremes,
    unit_slug,
)
from openswe_traces.synth.statefulness import score_unit

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


def test_rank_component_is_mean_entries() -> None:
    assert RANK_COMPONENT == "mean_distinct_entries"


def test_obfuscate_relpath_dir_renames() -> None:
    assert obfuscate_relpath("tikvrpc/interceptor/interceptor.go") == "wirerpc/interceptor/interceptor.go"
    assert obfuscate_relpath("internal/mockstore/mocktikv/mvcc.go") == "internal/mockstore/mockkv/mvcc.go"
    assert obfuscate_relpath("tikv/backoff.go") == "kvclient/backoff.go"
    assert obfuscate_relpath("internal/locate/region_cache.go") == "internal/locate/region_cache.go"


def test_unit_slug() -> None:
    assert unit_slug("NewPdOracle") == "new-pd-oracle"
    assert unit_slug("BatchGet") == "batch-get"


def test_is_used_skips_prior_ladder_entries() -> None:
    used = FeatureExcision(
        entry="LocateBucket",
        functions=("LocateBucket", "Contains"),
        files=("internal/locate/region_cache.go",),
        tests=("TestLocateBucket",),
        keep_interface=True,
        min_lines=92,
    )
    assert is_used(used) is True
    fresh = FeatureExcision(
        entry="NewPdOracle",
        functions=("NewPdOracle", "GetTimestamp", "Close"),
        files=("oracle/oracles/pd.go",),
        tests=("TestLocalOracle",),
        keep_interface=True,
        min_lines=116,
    )
    assert is_used(fresh) is False


def test_pick_extremes_top_and_bottom() -> None:
    from openswe_traces.synth.scoretest import RankedUnit

    def row(unit: str, score: float) -> RankedUnit:
        return RankedUnit(
            unit=unit,
            score=score,
            entry=unit,
            closure=(unit,),
            tests=("TestX",),
            files=(f"{unit}.go",),
            n_functions=3,
            n_lines=80,
            mean_calls=score,
            sequence_fraction=1.0,
            mean_distinct_entries=score,
            any_dynamic=False,
            n_tests=1,
        )

    ranked = [row(f"u{i}", float(10 - i)) for i in range(6)]
    picks = pick_extremes(ranked, n_each=2)
    assert [p.unit for p in picks] == ["u0", "u1", "u4", "u5"]


def test_collect_feature_excisions_fixture_band() -> None:
    _ensure_fixture_indexed()
    found = collect_feature_excisions(
        FIXTURE,
        min_functions=3,
        max_functions=8,
        min_lines=10,
        max_lines=None,
        package_local=True,
    )
    assert found
    for exc in found:
        assert 3 <= len(exc.functions) <= 8
        assert exc.min_lines >= 10
        assert exc.tests
        assert exc.entry[0].isupper()


def test_score_excision_mathx_in_package() -> None:
    from openswe_traces.synth.scoretest import locate_in_package_tests, score_excision

    _ensure_fixture_indexed()
    found = collect_feature_excisions(
        FIXTURE,
        min_functions=3,
        max_functions=8,
        min_lines=10,
        max_lines=None,
        package_local=True,
    )
    mathx = next((e for e in found if "mathx" in " ".join(e.files)), None)
    if mathx is None:
        mathx = next((e for e in found if e.entry in {"Add", "SumClamped", "Next"}), found[0])
    files, tests = locate_in_package_tests(FIXTURE, mathx)
    assert files
    assert tests
    ranked = score_excision(mathx, tree=FIXTURE)
    assert ranked is not None
    assert ranked.n_tests >= 1
    assert ranked.mean_distinct_entries >= 0.0
    # Existing fixture scorer still works independently.
    unit = score_unit(
        "mathx",
        test_files=[ROOT / "experiments/codegraph_bugs/fixture_host/mathx/mathx_test.go"],
        api=("Add", "SumClamped", "Clamp", "Next", "Put", "Get"),
    )
    assert unit.n_tests == 6
    assert Path(files[0]).suffix == ".go"


def test_score_units_l2_contract_and_patches() -> None:
    from pathlib import Path as P

    from openswe_traces.synth.affordance import with_no_web
    from openswe_traces.synth.ladder2 import _patch_touches_tests, unified_patch
    from openswe_traces.synth.obfuscate import NO_WEB_CLAUSE
    from openswe_traces.synth.scoretest import IMAGE_TAG, per_unit_dockerfile
    from openswe_traces.synth.scoretest_build import (
        NEXT_INSTRUCTION,
        _cheat_for,
        _stub_for,
        load_hidden,
        score_units,
    )

    units = score_units()
    assert {u.family for u in units} == {"delete-range", "batch-delete", "decode", "next"}
    for u in units:
        assert 3 <= len(u.functions) <= 8
        assert u.closure_lines >= 80
        assert u.entry[0].isupper()
        assert "go test" in u.instruction
        assert "-run" not in u.instruction
        for name, _sentence in u.coverage:
            assert name.startswith("Test")
            assert name not in u.instruction
        for sym in u.changed_symbols:
            assert sym not in u.instruction
        for fp in u.changed_files:
            assert P(fp).name.lower() not in u.instruction.lower()
        assert NO_WEB_CLAUSE in with_no_web(u.instruction)
        hidden = (
            P(__file__).resolve().parents[1]
            / "src/openswe_traces/synth/testdata/scoretest"
            / u.testdata_name
        )
        assert hidden.is_file()
        text = hidden.read_text(encoding="utf-8")
        assert "rand.New" in text
    docker = per_unit_dockerfile()
    assert f"FROM {IMAGE_TAG}" in docker
    assert "COPY src/" not in docker
    assert "excision.patch" in docker
    assert "patch -p1" in docker
    hidden_dr = load_hidden("delete_range_bb_prop_test.go")
    assert "deleteRangeCases = 10000" in hidden_dr
    assert "rand.New" in hidden_dr
    assert "TestDeleteRangeUnmentionedRandom" in hidden_dr
    hidden_bd = load_hidden("batch_delete_bb_prop_test.go")
    assert "batchDeleteCases = 10000" in hidden_bd
    assert "TestBatchDeleteUnmentionedRandom" in hidden_bd

    gold_next = "func (i *MemdbIterator) Next() error {\n\treturn i.advance()\n}\n"
    unit = next(u for u in units if u.family == "next")
    buggy = _stub_for(unit, gold_next)
    cheat = _cheat_for(unit, gold_next)
    gold_patch = unified_patch(unit.src_rel, buggy, gold_next)
    cheat_patch = unified_patch(unit.src_rel, buggy, cheat)
    for p in (gold_patch, cheat_patch):
        assert p
        assert not _patch_touches_tests(p)
    assert "nullAddr" in buggy
    assert NEXT_INSTRUCTION.startswith("# Missing behavior")
    assert "checkOnePC(" not in load_hidden("decode_bb_prop_test.go")
