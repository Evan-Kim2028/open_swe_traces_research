"""Smoke tests for the bank gate stack — no network, no repo tree needed."""
from __future__ import annotations

import json
from pathlib import Path

from openswe_traces import bank_gates as bg


def _unit(tmp: Path, name: str, instruction: str = "i",
          env: bool = False, hidden: bool = True, gold: bool = True) -> Path:
    d = tmp / name
    d.mkdir(parents=True)
    (d / "instruction.md").write_text(instruction)
    (d / "task.toml").write_text("[task]\n")
    if env:
        (d / "environment/src").mkdir(parents=True)
        (d / "environment/src/go.mod").write_text("module x\n")
    if hidden:
        (d / "tests/hidden").mkdir(parents=True)
        (d / "tests/hidden/t_test.go").write_text("package t\n")
    if gold:
        (d / "tests").mkdir(exist_ok=True)
        (d / "tests/gold.patch").write_text("patch\n")
    return d


def test_family_parsing():
    assert bg.family_of("helm-depresolver-L2") == "helm-depresolver"
    assert bg.family_of("helm-depresolver-L2auto3") == "helm-depresolver"
    assert bg.level_of("helm-depresolver-L2") == 2
    assert bg.level_of("chartloader") is None
    assert bg.stem_of("helm-depresolver-L2", "helm") == "depresolver"
    assert bg.stem_of("depresolver-L2", "helm") == "depresolver"
    assert bg.famkey("helm", "depresolver") == "helm/depresolver"


def test_repo_hints():
    assert bg.repo_of("helm-depresolver-L2") == "helm"
    assert bg.repo_of("jwtvalidate-L0", "sweep_nats") == "nats-server"
    assert bg.repo_of("idxdecode-L2", "sweep_gogit_L2") == "go-git"
    assert bg.repo_of("depresolver-L2", "helm") == "helm"
    assert bg.repo_of("xrepo-condreq-L0", "sweep_xrepo20") == "xrepo"


def test_find_units_dedupe(tmp_path):
    repo = tmp_path
    _unit(repo, "experiments/pipeline/tasks_composerver/helm/depresolver-L2")
    b = _unit(repo, "experiments/dose_response/sweep_helm2_L2/helm-depresolver-L2",
              env=True)
    b.joinpath("instruction.md").write_text("newer")
    import os
    import time
    t = time.time() + 100
    os.utime(b / "instruction.md", (t, t))

    refs = bg.find_units(repo)
    by_key = {}
    for rl in refs.values():
        for u in rl:
            by_key.setdefault(bg.ukey(u), []).append(u)
    assert "helm/depresolver@L2" in by_key
    assert len(by_key["helm/depresolver@L2"]) == 2
    canon = bg.pick_canonical(by_key["helm/depresolver@L2"])
    assert canon.has_env  # the sweep copy with env/src wins


def test_recommend():
    assert bg.recommend(["B10"], 0, "pass", False) == "drop"
    assert bg.recommend(["A12"], 0, "pass", False) == "repair"
    assert bg.recommend([], 3, "pass", False, 2) == "repair"   # gap>=2 at L2
    assert bg.recommend([], 3, "pass", False, 0) == "trial"    # L0: no contract
    assert bg.recommend([], 1, "pass", False, 2) == "trial"
    assert bg.recommend([], 0, "fail", True) == "repair"   # cold repo: A13 gates
    assert bg.recommend([], 0, "fail", False) == "trial"  # warm repo: advisory
    assert bg.recommend([], 0, "unjudged", True) == "repair"
    assert bg.recommend([], 0, "pass", False) == "trial"


def test_cg_gap_semantics():
    """gap = reached ∩ gold − implied, in-module symbols only."""
    from openswe_traces import cg_coverage as cc
    mp = "example.internal/mod"
    scan = {
        "module": mp,
        "tests": {
            "TestX": {
                "defs": [f"{mp}/pkg.Alpha", f"{mp}/pkg.(T).M", "strings.Cut"],
                "refs": [f"{mp}/pkg.B", f"{mp}/pkg.unexported",
                         f"{mp}/pkg.Orig"],
            },
            # contract-named original test: its reach is implied — but only
            # for symbols the hidden suite also reaches (candidates)
            "orig:TestOrig": {"defs": [f"{mp}/pkg.Orig"], "refs": []},
        },
        "gold_defs": [f"{mp}/pkg.Alpha", f"{mp}/pkg.B", f"{mp}/pkg.C",
                      f"{mp}/pkg.Orig", "strings.Cut"],
        "gold_refs": [f"{mp}/pkg.Helper", f"{mp}/pkg.unexported"],
        "excised": [],
    }
    text = "| `TestOrig` | does the orig thing |\nmentions HelperFuncName"
    sets = cc.compute_sets(scan, text)
    # reached∩gold = {Alpha, B, Orig, unexported}; strings.Cut filtered by
    # inmod; Helper not reached. Orig implied via named orig: test.
    assert set(sets["candidates"]) == {
        f"{mp}/pkg.Alpha", f"{mp}/pkg.B", f"{mp}/pkg.Orig",
        f"{mp}/pkg.unexported"}
    assert sets["gap_deterministic"] == [
        f"{mp}/pkg.Alpha", f"{mp}/pkg.B", f"{mp}/pkg.unexported"]
    # literal-mention clears a candidate whose leaf appears in the prose
    sets2 = cc.compute_sets(scan, text + " and calls Alpha")
    assert sets2["gap_deterministic"] == [f"{mp}/pkg.B", f"{mp}/pkg.unexported"]


def test_confusion():
    flag = {"a": True, "b": True, "c": False}
    outc = {"a": False, "b": True, "c": False}
    c = bg._confusion(flag, outc)
    assert (c["tp"], c["fp"], c["fn"], c["tn"]) == (1, 1, 1, 0)
    assert abs(c["precision"] - 0.5) < 1e-9


def test_ledger_parsing(tmp_path):
    job = tmp_path / "experiments/dose_response/jobs/sweep_x/unit-L2__a1"
    job.mkdir(parents=True)
    (job / "result.json").write_text(json.dumps({
        "task_name": "helm-depresolver-L2",
        "task_id": {"path": "experiments/dose_response/sweep_helm2_L2/helm-depresolver-L2"},
        "verifier_result": {"rewards": {"reward": 0}},
        "agent_result": {"n_input_tokens": 100, "n_output_tokens": 50},
        "started_at": "2026-09-20T00:00:00Z",
    }))
    trials = bg.load_ledger(tmp_path)
    assert len(trials) == 1
    assert trials[0].family == "helm/depresolver"
    assert trials[0].tokens == 150
    fams = bg.family_outcomes(trials)
    assert fams["helm/depresolver"].escalated
    assert not fams["helm/depresolver"].flipped
