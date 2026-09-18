from __future__ import annotations

from pathlib import Path

from openswe_traces.synth.affordance import with_no_web
from openswe_traces.synth.ladder2 import (
    CHAIN_INSTRUCTION,
    alt_chain,
    cheat_chain,
    render_measure_gold_sh,
    replace_go_func,
    stub_chain,
    unified_patch,
    unit_specs,
)
from openswe_traces.synth.obfuscate import NO_WEB_CLAUSE

GOLD_WRAP = """package interceptor

func (c *RPCInterceptorChain) Wrap(next RPCInterceptorFunc) RPCInterceptorFunc {
	for n := len(c.chain) - 1; n >= 0; n-- {
		next = c.chain[n].Wrap(next)
	}
	return next
}

func (c *RPCInterceptorChain) Link(it RPCInterceptor) *RPCInterceptorChain {
	for i := range c.chain {
		if c.chain[i].Name() == it.Name() {
			c.chain = append(c.chain[:i], c.chain[i+1:]...)
			break
		}
	}
	c.chain = append(c.chain, it)
	return c
}
"""


def test_unit_mix_and_band() -> None:
    units = unit_specs()
    assert len(units) == 6
    kinds = [u.kind for u in units]
    assert kinds.count("spec-bb") == 2
    assert kinds.count("property") == 2
    assert kinds.count("dynamic") == 2
    families = {u.family for u in units}
    assert families == {
        "spec-bb-chain",
        "spec-bb-bucket",
        "property-policy",
        "property-1pc",
        "dynamic-snapshot",
        "dynamic-latch",
    }
    for u in units:
        assert 3 <= len(u.functions) <= 8
        assert 80 <= u.closure_lines <= 400
        assert u.entry[0].isupper()
        assert u.existing_tests
        assert "go test" in u.instruction
        assert "-run" not in u.instruction
        for name, _sentence in u.coverage:
            assert name.startswith(("Test", "Benchmark"))
            if name.startswith("Test"):
                assert name not in u.instruction
        for sym in u.changed_symbols:
            assert sym not in u.instruction
        for fp in u.changed_files:
            assert Path(fp).name.lower() not in u.instruction.lower()
        text = with_no_web(u.instruction)
        assert NO_WEB_CLAUSE in text
        hidden = (Path(__file__).resolve().parents[1] / "src/openswe_traces/synth/testdata/ladder2" / u.testdata_name)
        assert hidden.is_file()


def test_replace_and_patches_skip_tests() -> None:
    buggy = stub_chain(GOLD_WRAP)
    assert "return next" in buggy
    gold_patch = unified_patch("wirerpc/interceptor/interceptor.go", buggy, GOLD_WRAP)
    cheat_patch = unified_patch("wirerpc/interceptor/interceptor.go", buggy, cheat_chain(GOLD_WRAP))
    alt_patch = unified_patch("wirerpc/interceptor/interceptor.go", buggy, alt_chain(GOLD_WRAP))
    for p in (gold_patch, cheat_patch, alt_patch):
        assert p
        assert "_test.go" not in p
        assert "diff --git a/wirerpc/interceptor/interceptor.go" in p
    new = replace_go_func(
        GOLD_WRAP,
        "func (c *RPCInterceptorChain) Wrap(next RPCInterceptorFunc) RPCInterceptorFunc {",
        "func (c *RPCInterceptorChain) Wrap(next RPCInterceptorFunc) RPCInterceptorFunc {\n\treturn next\n}",
    )
    assert "for n :=" not in new


def test_measure_gold_sh() -> None:
    empty = render_measure_gold_sh(perf_bench="", pkg="latch", gold_rel="x.go", gold_copy="g.go")
    assert "no timing gate" in empty
    sh = render_measure_gold_sh(
        perf_bench="BenchmarkSnapshotGet",
        pkg="internal/unionstore",
        gold_rel="internal/unionstore/memdb_snapshot.go",
        gold_copy="gold_memdb_snapshot.go",
    )
    assert "BenchmarkSnapshotGet" in sh
    assert "limit=" in sh
    assert "load_avg" in sh
    assert CHAIN_INSTRUCTION.startswith("# Missing behavior")
