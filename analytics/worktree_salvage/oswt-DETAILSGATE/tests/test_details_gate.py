from __future__ import annotations

import json
from pathlib import Path

from openswe_traces.details_gate import (
    DetailLine,
    LineJudgement,
    collect_trial_verdicts,
    find_annotation,
    gate_passed,
    gate_unit,
    load_records,
    parse_answer,
    parse_details,
    report,
    run_batch,
    solver_visible_diff,
    unit_verdict,
    validate_outcomes,
    write_gate_file,
)

DETAILS_SAMPLE = """1. First commitment stays. Inferable: yes.
2. Second commitment wraps
   onto a second line. Not inferable — undocumented.
3. Third commitment. Inferable: doc — from package docs.
4. Fourth commitment, partially inferable from callers.
5. Fifth commitment with no annotation.
"""


def test_parse_details_entries_and_annotations():
    lines = parse_details(DETAILS_SAMPLE)
    assert [d.n for d in lines] == [1, 2, 3, 4, 5]
    assert lines[0].author_value == "yes"
    assert lines[1].author_value == "no"
    assert lines[1].commitment.startswith("Second commitment wraps onto a second line")
    assert "Not inferable" not in lines[1].commitment
    assert lines[2].author_value == "doc"
    assert lines[3].author_value == "partially"
    assert lines[4].author_value == "none"
    assert lines[4].author_raw == ""


def test_find_annotation_bare_inferable_in_first_half_is_content():
    val, raw, _ = find_annotation("The inferable flag is stored. Some long tail without judgement.")
    assert val is None and raw is None


def test_find_annotation_prose_tail():
    val, raw, _ = find_annotation("Behaviour X holds across retries. Not inferable.")
    assert val == "no"
    assert "Not inferable" in raw


def test_find_annotation_free_text_partially():
    val, _, _ = find_annotation("Some rule. Inferable: the variable is exported — partially")
    assert val == "partially"


def test_solver_visible_diff_strips_gold():
    patch = """diff --git a/lib/foo.go b/lib/foo.go
--- a/lib/foo.go
+++ b/lib/foo.go
@@ -1,4 +1,4 @@
 func Foo() int {
-    return computeSecret()
+    return 0 // stub
 }
diff --git a/lib/foo_test.go b/lib/foo_test.go
--- a/lib/foo_test.go
+++ b/lib/foo_test.go
@@ -1,3 +0,0 @@
-func TestFoo() { want := 42 }
"""
    vis = solver_visible_diff(patch)
    assert "computeSecret" not in vis  # removed impl line gone
    assert "stub" in vis  # added stub survives
    assert "TestFoo" not in vis  # whole test-file hunk gone
    assert "foo.go" in vis


def test_parse_answer_and_coercions():
    lines = [DetailLine(n=i, raw_text=f"c{i}", commitment=f"c{i}", author_value="none", author_raw="") for i in (1, 2, 3)]
    ans = "1|ARBITRARY|grade|literal only\n2|DERIVABLE|grade-shape-only|has literal\n3|COUNTER|drop|oops"
    j = parse_answer(ans, lines)
    assert j[1].action == "grade-shape-only"  # ARBITRARY|grade repaired
    assert j[2].action == "grade-shape-only"  # DERIVABLE|shape-only is real
    assert j[3].action == "grade"  # COUNTER|drop repaired
    assert j[2].kind == "DERIVABLE"


def test_parse_answer_missed_line_defaults():
    lines = [DetailLine(n=1, raw_text="a", commitment="a", author_value="none", author_raw=""),
             DetailLine(n=2, raw_text="b", commitment="b", author_value="none", author_raw="")]
    j = parse_answer("1|DERIVABLE|grade|ok", lines)
    assert j[2].kind == "UNKNOWN"
    assert j[2].action == "drop"


def test_unit_verdicts():
    lines = [DetailLine(n=i, raw_text="x", commitment="x", author_value="none", author_raw="") for i in (1, 2, 3, 4)]
    all_grade = {d.n: LineJudgement(kind="DERIVABLE", action="grade") for d in lines}
    assert unit_verdict(lines, all_grade, model_ok=True) == "pass"
    assert unit_verdict(lines, all_grade, model_ok=False) == "fail"
    three_arb = {d.n: LineJudgement(kind="ARBITRARY", action="grade-shape-only") for d in lines[:3]}
    three_arb[4] = LineJudgement(kind="DERIVABLE", action="grade")
    assert unit_verdict(lines, three_arb, model_ok=True) == "low-discrimination"
    all_drop = {d.n: LineJudgement(kind="ARBITRARY", action="drop") for d in lines}
    assert unit_verdict(lines, all_drop, model_ok=True) == "fail"


def _mk_author(tmp: Path, details: str | None = DETAILS_SAMPLE) -> Path:
    author = tmp / "repo" / "unit" / "_author"
    author.mkdir(parents=True)
    if details is not None:
        (author / "DETAILS.md").write_text(details)
    return author


def test_gate_passed_states(tmp_path):
    author = _mk_author(tmp_path)
    assert not gate_passed(author)  # DETAILS present, no gate file
    (author / "details_gate.json").write_text("not json")
    assert not gate_passed(author)
    (author / "details_gate.json").write_text(json.dumps({"verdict": "low-discrimination"}))
    assert not gate_passed(author)
    (author / "details_gate.json").write_text(json.dumps({"verdict": "pass"}))
    assert gate_passed(author)


def test_gate_passed_no_details_is_true(tmp_path):
    author = _mk_author(tmp_path, details=None)
    assert gate_passed(author)


def test_gate_unit_fake_ask(tmp_path):
    author = _mk_author(tmp_path)
    seen = {}

    def fake_ask(prompt, cache_key=None):
        seen["prompt"] = prompt
        return "\n".join(
            f"{i}|DERIVABLE|grade|reason{i}" for i in range(1, 6)
        )

    g = gate_unit("repo", "unit", author, ask_fn=fake_ask)
    assert g.verdict == "pass"
    assert all(g.judgements[d.n].kind == "DERIVABLE" for d in g.lines)
    assert "Inferable: yes" not in seen["prompt"]  # author labels stripped
    path = write_gate_file(g)
    data = json.loads(path.read_text())
    assert data["family"] == "repo-unit"
    assert data["counts"]["DERIVABLE"] == 5


def test_gate_unit_error_answer(tmp_path):
    author = _mk_author(tmp_path)
    g = gate_unit("repo", "unit", author, ask_fn=lambda *a, **k: "__ERROR__ boom")
    assert g.verdict == "fail"
    assert g.error.startswith("__ERROR__")


def test_collect_trial_verdicts(tmp_path):
    d = tmp_path / "run" / "fam-L0__trial1"
    d.mkdir(parents=True)
    (d / "result.json").write_text(json.dumps({
        "task_name": "fam-L0",
        "verifier_result": {"rewards": {"reward": 0.0}},
    }))
    d2 = tmp_path / "run" / "fam-L0__trial2"
    d2.mkdir(parents=True)
    (d2 / "result.json").write_text(json.dumps({
        "task_name": "fam-L0",
        "verifier_result": {"rewards": {"reward": 1.0}},
    }))
    d3 = tmp_path / "run" / "fam-L2__trial1"
    d3.mkdir(parents=True)
    (d3 / "result.json").write_text(json.dumps({
        "task_name": "fam-L2",
        "verifier_result": {"rewards": {"reward": 0.0}},
    }))
    # a no-reward trial is ignored
    d4 = tmp_path / "run" / "fam-L2__trial2"
    d4.mkdir(parents=True)
    (d4 / "result.json").write_text(json.dumps({
        "task_name": "fam-L2",
        "verifier_result": {"rewards": {}},
    }))
    v = collect_trial_verdicts(tmp_path)
    assert v["fam"]["L0"] == "pass"  # any 1.0 -> pass
    assert v["fam"]["L2"] == "fail"  # all real trials 0 -> fail


def test_validate_outcomes_precision_recall():
    recs = [
        {"family": "a", "verdict": "pass", "counts": {"lines": 4, "ARBITRARY": 2, "drop": 1}},
        {"family": "b", "verdict": "pass", "counts": {"lines": 4, "ARBITRARY": 0, "drop": 0}},
        {"family": "c", "verdict": "pass", "counts": {"lines": 4, "ARBITRARY": 3, "drop": 0}},
        {"family": "d", "verdict": "pass", "counts": {"lines": 4, "ARBITRARY": 0, "drop": 0}},
    ]
    verdicts = {
        "a": {"L0": "fail", "L2": "fail"},  # double fail, flagged
        "b": {"L0": "fail", "L2": "fail"},  # double fail, not flagged
        "c": {"L0": "pass", "L2": "fail"},  # flagged, not double
        "d": {"L0": "fail", "L2": "pass"},
    }
    out = validate_outcomes(recs, verdicts)
    assert out["n_both"] == 4
    assert out["n_double_fail"] == 2
    assert out["double_fail_rate"] == 0.5
    any_arb = out["rules"]["any_ARBITRARY"]
    assert any_arb["n_flagged"] == 2
    assert any_arb["tp"] == 1
    assert any_arb["precision"] == 0.5
    assert any_arb["recall"] == 0.5


def test_report_confusion_shape():
    recs = [{
        "family": "x",
        "verdict": "pass",
        "counts": {"lines": 2},
        "lines": [
            {"kind": "ARBITRARY", "author_inferable": "no"},
            {"kind": "DERIVABLE", "author_inferable": "yes"},
        ],
    }]
    rep = report(recs, {})
    assert rep["confusion"]["ARBITRARY"]["no"] == 1
    assert rep["confusion"]["DERIVABLE"]["yes"] == 1


def test_run_batch_incremental_and_resume(tmp_path, monkeypatch):
    author = _mk_author(tmp_path)
    out = tmp_path / "gate.jsonl"

    calls = {"n": 0}

    class FakeComposer:
        @staticmethod
        def ask(prompt, cache_key=None):
            calls["n"] += 1
            return "1|DERIVABLE|grade|ok\n2|DERIVABLE|grade|ok\n3|DERIVABLE|grade|ok\n4|DERIVABLE|grade|ok\n5|DERIVABLE|grade|ok"

    import sys
    monkeypatch.setitem(sys.modules, "ask_composer", FakeComposer)
    units = [("repo", "unit", author)]
    res = run_batch(units, out, workers=1)
    assert calls["n"] == 1
    assert len(res) == 1
    assert out.read_text().count("\n") == 1
    assert (author / "details_gate.json").is_file()
    # resume: family already in JSONL -> skipped, no new call
    res2 = run_batch(units, out, workers=1)
    assert calls["n"] == 1
    assert res2 == []


def test_load_records_skips_bad_lines(tmp_path):
    p = tmp_path / "r.jsonl"
    p.write_text('{"family":"a"}\nnot-json\n{"family":"b"}\n')
    recs = load_records(p)
    assert [r["family"] for r in recs] == ["a", "b"]


# ---------------------------------------------------------------------------
# pipeline integration: verification and packaging refuse an ungated unit
# ---------------------------------------------------------------------------

import pytest
from test_pipeline import FakeRunner, _cfg, _ok_proof, _write_unit


def _unit_with_details(tmp_path: Path) -> Path:
    batch = tmp_path / "work" / "mathx" / "author_batch"
    author = _write_unit(batch, "unit0")
    (author / "DETAILS.md").write_text("1. Add returns the sum. Inferable: yes.\n")
    return author


def test_verifier_rejects_ungated_details(tmp_path):
    cfg = _cfg(tmp_path)
    from openswe_traces.pipeline.state import REJECTED, PipelineStore
    from openswe_traces.pipeline.verifier import VerifierReject, run_verifier

    store = PipelineStore(cfg.state_db)
    _unit_with_details(tmp_path)
    runner = FakeRunner(cfg.work_dir, n=1)
    with pytest.raises(VerifierReject) as exc:
        run_verifier("mathx", "unit0", cfg, store, runner=runner, prove=_ok_proof)
    assert exc.value.rule_id == "DETAILS"
    rows = store.list_units("mathx")
    assert rows[0]["status"] == REJECTED
    assert rows[0]["rejected_rule"] == "DETAILS"
    assert runner.calls == []
    store.close()


def test_verifier_accepts_passing_gate(tmp_path):
    cfg = _cfg(tmp_path)
    from openswe_traces.pipeline.state import PipelineStore
    from openswe_traces.pipeline.verifier import run_verifier

    store = PipelineStore(cfg.state_db)
    author = _unit_with_details(tmp_path)
    (author / "details_gate.json").write_text(json.dumps({"verdict": "pass"}))
    runner = FakeRunner(cfg.work_dir, n=1)
    out = run_verifier("mathx", "unit0", cfg, store, runner=runner, prove=_ok_proof)
    assert out["proof"]["ok"] is True
    assert store.list_units("mathx")[0]["status"] == "verified"
    store.close()


def test_package_levels_rejects_ungated_details(tmp_path):
    cfg = _cfg(tmp_path)
    from openswe_traces.pipeline.package import package_levels

    _unit_with_details(tmp_path)
    with pytest.raises(FileNotFoundError, match="details_gate"):
        package_levels("mathx", "unit0", cfg, levels=[0, 2])
