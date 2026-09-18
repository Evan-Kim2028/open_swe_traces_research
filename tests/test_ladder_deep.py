from __future__ import annotations

import re

from openswe_traces.synth.affordance import instruction_for_level
from openswe_traces.synth.ladder_deep import unit_specs
from openswe_traces.synth.obfuscate import NO_WEB_CLAUSE


def test_deep_units_and_gaps() -> None:
    units = unit_specs()
    assert {u.family for u in units} == {
        "dynamic-pipeline",
        "spec-reimpl-bb",
        "spec-bb-chain",
        "dynamic-latch",
    }
    held = {u.family for u in units if not u.flip_at_a0}
    controls = {u.family for u in units if u.flip_at_a0}
    assert held == {"dynamic-pipeline", "spec-reimpl-bb"}
    assert controls == {"spec-bb-chain", "dynamic-latch"}
    for unit in units:
        assert unit.gap.omitted
        assert ":" in unit.gap.discoverable
        assert unit.gap.catcher_test.startswith("Test")
        assert unit.gap.catcher_file.endswith("_test.go")
        for instr in (unit.instruction_a1, unit.instruction_a2):
            assert "expected" in instr.lower() or "actual" in instr.lower()
            assert "go test" in instr
            assert "-run" not in instr
            text = instruction_for_level(instr, [], -1)
            assert NO_WEB_CLAUSE in text
            for name in unit.changed_symbols:
                assert not re.search(rf"\b{re.escape(name)}\b", instr)
            for fp in unit.changed_files:
                assert fp.lower() not in instr.lower()
        assert unit.gap.catcher_test not in unit.instruction_a1
        assert unit.gap.catcher_test not in unit.instruction_a2
        assert unit.source.is_dir()
