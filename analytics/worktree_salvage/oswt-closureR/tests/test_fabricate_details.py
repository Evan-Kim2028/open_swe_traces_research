"""Tests for the detail-count x verification-affordance dial (closure-R).

Covers the detail registry (drawing, independence, discrimination), the v
knob (worked examples present at v=high, absent for drawn details at v=low),
the manifest details block, and the B3 not-a-complete-spec guarantee.  The
slow Go-level proof/audit machinery is exercised on one small cell.
"""

from __future__ import annotations

import json
import re
import shutil
from pathlib import Path

import pytest

from openswe_traces.synth import fab_fairness
from openswe_traces.synth import fabricate as F


def _require_go() -> None:
    if shutil.which("go") is None:
        pytest.skip("go toolchain not on PATH")


def _fixed_cfg(domain: str = "store", seed: int = 11) -> F.Config:
    return F.Config(
        domain=domain,
        seed=seed,
        n_funcs=22,
        target_ratio=0.0,
        E=9,
        K=2,
        canon_mode="pipe",
        canon_in_S=True,
        misc_in_S=True,
        place_in_S=domain == "sched",
        consumers=(),
        fill=0,
        consts=F.DOMAINS[domain]["consts"](seed),
    )


def test_registry_has_enough_details_for_d8() -> None:
    assert len(F.DETAILS["store"]) >= 8
    assert len(F.DETAILS["sched"]) >= 8


def test_draw_is_deterministic_and_independent_per_cell() -> None:
    a = F.draw_details("store", 11, 4, present=set(F.STORE_ENTRIES[:9]))
    b = F.draw_details("store", 11, 4, present=set(F.STORE_ENTRIES[:9]))
    assert a == b
    # each (seed, d) cell draws afresh: d=2 is not a subset of d=4
    two = set(F.draw_details("store", 11, 2, present=set(F.STORE_ENTRIES[:9])))
    four = set(F.draw_details("store", 11, 4, present=set(F.STORE_ENTRIES[:9])))
    assert len(two) == 2 and len(four) == 4
    # a different seed draws a different set (with overwhelming probability)
    other = set(F.draw_details("store", 23, 4, present=set(F.STORE_ENTRIES[:9])))
    assert len(other) == 4


def test_draw_respects_config_presence() -> None:
    # E=4 exposes only NewStore/Put/Get/BucketOf: no Tags/Prune/Count details
    present = set(F.STORE_ENTRIES[:4])
    ids = F.draw_details("store", 11, 0, present=present)
    assert ids  # still drawable
    for did in ids:
        assert set(F.DETAILS["store"][did]["requires"]).issubset(present)


def test_every_detail_trap_discriminates_generation_time() -> None:
    # The smoke generators raise if no discriminating input exists for a seed;
    # exercising all store details on two seeds proves each trap is detectable
    # by construction (v=high affordance) and that hidden inputs exceed the
    # shown examples (B3).
    for seed in (11, 42):
        cfg = _fixed_cfg(seed=seed)
        for did, entry in F.DETAILS["store"].items():
            out = entry["gen"](cfg)
            smoke, hidden = out["smoke"], out["hidden"]
            assert entry["smoke_name"] in smoke
            hidden_src, _ = hidden
            assert entry["hidden_name"] in hidden_src
            if did in ("bucket_reduction", "tag_normalization", "overwrite_semantics"):
                assert "rand.New" in hidden_src  # property inputs beyond examples


def test_v_knob_controls_worked_examples(tmp_path: Path) -> None:
    cfg = _fixed_cfg()
    drawn = {"bucket_reduction", "tag_normalization"}
    high = F._in_tree_tests("store", cfg, drawn, "high")
    low = F._in_tree_tests("store", cfg, drawn, "low")
    assert "TestSmoke_bucket_reduction" in high
    assert "TestSmoke_tag_normalization" in high
    # drawn details' worked examples are dropped at v=low ...
    assert "TestSmoke_bucket_reduction" not in low
    assert "TestSmoke_tag_normalization" not in low
    # ... while non-drawn details stay visible at both levels
    assert "TestSmoke_reserved_key" in low
    assert "TestSmoke_reserved_key" in high


def test_manifest_details_block(tmp_path: Path) -> None:
    _require_go()
    cfg = _fixed_cfg()
    out = tmp_path / "u"
    r = F.generate_unit(
        seed=11, n_funcs=22, ratio=0.0, out=out, domain="store", d=4, v="low",
        cfg=cfg, proof=True, scratch_root=tmp_path / "proofs",
    )
    assert r.proof["ok"]
    manifest = json.loads((out / "manifest.json").read_text(encoding="utf-8"))
    assert manifest["dial"]["d_achieved"] == 4
    assert manifest["dial"]["v"] == "low"
    assert manifest["dial"]["constant_across_grid"]["S_size"] == len(r.stats.S)
    details = manifest["details"]
    assert len(details) == 4
    for d in details:
        assert d["stated_in_bugreport"] is True
        assert d["checkable_from_repro"] is False
        assert d["prose"] in (out / "_author" / "bugreport.md").read_text(encoding="utf-8")
        for t in d["hidden_tests_covering"]:
            assert re.search(rf"^func {t}\s*\(", (out / "hidden" / "store_bb_test.go").read_text(encoding="utf-8"), re.MULTILINE)


def test_fairness_including_b3_and_details(tmp_path: Path) -> None:
    _require_go()
    for v in ("low", "high"):
        cfg = _fixed_cfg(seed=23)
        out = tmp_path / f"u-{v}"
        r = F.generate_unit(
            seed=23, n_funcs=22, ratio=0.0, out=out, domain="store", d=6, v=v,
            cfg=cfg, proof=True, scratch_root=tmp_path / "proofs",
        )
        assert r.proof["ok"]
        a = fab_fairness.audit_unit(out)
        names = {c.name: c.passed for c in a.checks}
        assert names["details_manifest"], {c.name: c.detail for c in a.checks if not c.passed}
        assert names["b3_not_complete_spec"], {c.name: c.detail for c in a.checks if not c.passed}
        assert names["bugreport_ceiling"]


def test_per_detail_vector_emitted_locally(tmp_path: Path) -> None:
    """The hidden suite writes one JSONL row per drawn detail (+guard) to the
    path in FAB_DETAILS_LOG — the verifier vector is parseable and complete."""
    _require_go()
    import json
    import subprocess

    cfg = _fixed_cfg(seed=11)
    out = tmp_path / "u"
    F.generate_unit(
        seed=11, n_funcs=22, ratio=0.0, out=out, domain="store", d=4, v="high",
        cfg=cfg, proof=False,
    )
    tree = tmp_path / "goldtree"
    drawn = set(F.draw_details("store", 11, 4, present=set(F.STORE_ENTRIES[:9])))
    F._write_tree(tree, F.render_module_files("store", cfg, drawn, "high"))
    shutil.copy2(out / "hidden" / "store_bb_test.go", tree / "store_bb_test.go")
    env = {**F.GO_ENV, "FAB_DETAILS_LOG": str(tmp_path / "details.json")}
    p = subprocess.run(
        ["go", "test", "-count=1", "-timeout", "3m", "./..."],
        cwd=tree, capture_output=True, text=True, env=env, check=False,
    )
    assert p.returncode == 0, (p.stdout + p.stderr)[-1500:]
    lines = [json.loads(ln) for ln in (tmp_path / "details.json").read_text().splitlines() if ln.strip()]
    ids = {ln["id"] for ln in lines}
    manifest = json.loads((out / "manifest.json").read_text(encoding="utf-8"))
    assert "guard" in ids
    assert {d["id"] for d in manifest["details"]} <= ids
    assert all(ln["pass"] is True for ln in lines)  # gold passes everything


def test_trap_audit_single_detail(tmp_path: Path) -> None:
    """One drawn detail: trap invisible at v=low, visible at v=high, caught by
    its own hidden test, and no other test depends on it."""
    _require_go()
    cfg = _fixed_cfg(seed=11)
    out = tmp_path / "u"
    F.generate_unit(
        seed=11, n_funcs=22, ratio=0.0, out=out, domain="store", d=1, v="low",
        cfg=cfg, proof=False,
    )
    drawn = set(F.draw_details("store", 11, 1, present=set(F.STORE_ENTRIES[:9])))
    assert len(drawn) == 1
    audit = F.audit_unit_details(out, cfg, drawn, "low", tmp_path / "proofs")
    assert audit["ok"], {k: v for k, v in audit.items() if k != "ok"}
