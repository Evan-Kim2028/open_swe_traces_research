"""Unit tests for the repair loop — no docker, no network."""

from __future__ import annotations

import json

from openswe_traces.gate import repair_loop as rl
from openswe_traces.gate.shadow import parse_fail_blocks

GO_TEST_LOG = """--- FAIL: TestRIContractTable (0.02s)
    repindex_bb_prop_test.go:166: duplicate add did not append: n=1
    repindex_bb_prop_test.go:174: unparseable version should error
--- FAIL: TestRIAddSort (3.11s)
    repindex_bb_prop_test.go:269: case 5: beta not sorted desc: [0.0.1 0.0.3-beta.2]
--- PASS: TestRIMerge (0.31s)
FAIL\texample.internal/helm/pkg/repo/v1\t4.581s
FAIL
"""


def test_parse_fail_blocks():
    blocks = parse_fail_blocks(GO_TEST_LOG)
    assert set(blocks) == {"TestRIContractTable", "TestRIAddSort"}
    assert blocks["TestRIContractTable"][0].endswith("n=1")
    assert "not sorted desc" in blocks["TestRIAddSort"][0]
    assert "TestRIMerge" not in blocks


def test_parse_fail_blocks_subtests():
    log = "--- FAIL: TestX/sub1 (0.01s)\n    x_test.go:5: boom\n"
    assert parse_fail_blocks(log) == {"TestX/sub1": ["x_test.go:5: boom"]}


def test_contract_head_tail():
    instr = "# Contract (L2) — x\n\nprose here.\n\nReproduce with:\n\ntests/test.sh\n\nIMPORTANT: no web.\n"
    head, tail = rl.contract_head_tail(instr)
    assert head.startswith("# Contract") and "prose" in head
    assert "Reproduce with:" not in head
    assert tail.startswith("\nReproduce with:") and "no web" in tail


def test_contract_head_tail_bugreport():
    instr = "# Contract\n\nrows.\n\n# Bug report\n\nsymptom.\n\nReproduce with:\n\nx\n"
    head, tail = rl.contract_head_tail(instr)
    assert "# Bug report" not in head
    assert "symptom" in tail and "Reproduce with:" in tail


def test_literal_budget():
    hidden = 'if got != "0.0.3-beta.2" { t.Fatal() } // sha256:deadbeef'
    text = 'uses `0.0.3-beta.2` and `invented-thing` and `sha256:deadbeef`'
    n, g, frac = rl.literal_budget(text, hidden)
    assert n == 3 and g == 2 and abs(frac - 2 / 3) < 1e-9


def test_check_repair_literal_cap():
    hidden = '"aaa1" "bbb2" "ccc3"'
    old = '`aaa1` `zzz9`\n\nReproduce with:\n\ntests/test.sh\n'
    # adding one grounded literal and dropping nothing: count rises -> reject
    new = '`aaa1` `zzz9` `bbb2`\n\nReproduce with:\n\ntests/test.sh\n'
    v = rl.check_repair(new.split("Reproduce")[0], old, new, [], set(), hidden)
    assert any("literal count rose" in x for x in v)
    # dropping the ungrounded literal keeps count flat and raises frac -> ok
    pad = "Behavioural prose padding so the head is non-trivial. " * 4
    new2 = f'`aaa1` `bbb2`\n\n{pad}\n\nReproduce with:\n\ntests/test.sh\n'
    v2 = rl.check_repair(new2.split("Reproduce")[0], old, new2, [], set(), hidden)
    assert not v2


def test_check_repair_hidden_names_and_scrub():
    old = "prose\n\nReproduce with:\n\ntests/test.sh\n"
    new = 'the call sorts. `TestRIFoo` rows.\n\nReproduce with:\n\ntests/test.sh\n'
    v = rl.check_repair(new.split("Reproduce")[0], old, new, [],
                        {"TestRIFoo"}, "")
    assert any("hidden test names" in x for x in v)
    assert any("the call" in x for x in v)


def test_unified_diff():
    d = rl.unified_diff("a\nb\n", "a\nc\n")
    assert "-b" in d and "+c" in d and "a/instruction.md" in d


def test_stage_name(tmp_path):
    sweep = tmp_path / "sweep_goa_L2"
    (sweep / "exprhash-L2").mkdir(parents=True)
    assert rl.stage_name(sweep / "exprhash-L2") == "goa-exprhash-L2loop"
    tasks = tmp_path / "tasks_batch2" / "bbolt"
    (tasks / "page-L2").mkdir(parents=True)
    assert rl.stage_name(tasks / "page-L2") == "bbolt-page-L2loop"
    climb = tmp_path / "sweep_climb_L3"
    (climb / "helm-repindex-L3").mkdir(parents=True)
    assert rl.stage_name(climb / "helm-repindex-L3") == "helm-repindex-L2loop"


def test_attribution_prompt_lists_each_failure():
    contract = "# Contract\n\nsome prose\n"
    blocks = {"TestA": ["msg one"], "TestB": ["msg two", "msg three"]}
    bodies = {"TestA": "func TestA(t) {}", "TestB": "func TestB(t) {}"}
    p = rl.attribution_prompt(contract, blocks, bodies)
    assert p.count("TEST `") == 3
    assert "some prose" in p
    assert "func TestB" in p


def test_attribute_fallback_on_bad_judge(monkeypatch):
    class R:
        text = None
        model = "m"
        requests = 1
        tokens = 5
        error = "boom"
        cache_hit = False

    monkeypatch.setattr(rl, "llm_call", lambda *a, **k: R())
    attrs, _res = rl.attribute("contract", {"TestX": ["m1"]}, [], "k", "t")
    assert attrs[0].verdict == "ambiguous"
    assert attrs[0].evidence == "attribution unavailable"


def test_make_patch(tmp_path):
    old = tmp_path / "old"
    new = tmp_path / "new"
    (old / "pkg").mkdir(parents=True)
    (new / "pkg").mkdir(parents=True)
    (old / "pkg" / "a.go").write_text("func A() {}\n")
    (new / "pkg" / "a.go").write_text("func A() { panic(1) }\n")
    patch = rl.make_patch(old, new)
    assert "diff --git a/pkg/a.go b/pkg/a.go" in patch
    assert "-func A() {}" in patch and "+func A() { panic(1) }" in patch


def test_result_row_and_done(tmp_path):
    r = rl.LoopResult("u1", "converged_fixed_point", [], "", 3, 100, 1.0)
    row = rl._result_row(r)
    assert row["outcome"] == "converged_fixed_point"
    path = tmp_path / "results.jsonl"
    path.write_text(json.dumps(row) + "\n" + json.dumps({"unit": "u2", "outcome": "error"}) + "\n")
    assert rl.done_units(path) == {"u1"}  # error retries


def test_cheat_prompt_flips_intent():
    ctx_prompt = ("You are re-implementing excised functions in a Go package. "
                  "You are given (1) the task CONTRACT ...")
    p = rl.cheat_prompt(ctx_prompt)
    assert "CHEAT" in p and "MINIMUM real work" in p
    assert "re-implementing" not in p.split("Your goal")[0]
