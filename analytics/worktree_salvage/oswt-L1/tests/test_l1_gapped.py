"""L1 gapped-contract mapping matches the packaged L2 instructions."""

from __future__ import annotations

import pytest

from openswe_traces.synth.affordance import omit_invariant
from openswe_traces.synth.l1_gapped import UNITS


@pytest.mark.parametrize("gap", UNITS, ids=lambda g: f"{g.repo}/{g.unit}")
def test_omit_strings_match_l2_instruction(gap) -> None:
    if not gap.l2_dir.is_dir():
        pytest.skip(f"L2 not packaged: {gap.l2_dir}")
    text = (gap.l2_dir / "instruction.md").read_text(encoding="utf-8")
    for kind, row, prose in (
        ("binding", gap.binding_row, gap.binding_prose),
        ("other", gap.other_row, gap.other_prose),
    ):
        gapped, omitted = omit_invariant(text, row_contains=row, prose=prose)
        assert prose not in gapped, kind
        assert gapped != text, kind
        if row:
            assert omitted.table_row
            assert row.strip("`") in omitted.original_test or row in omitted.table_row
            assert omitted.table_row not in gapped
        else:
            assert omitted.table_row == ""
        # Reproduce clause and no-web stay.
        assert "tests/test.sh" in gapped
        assert "Do NOT use web search" in gapped
