from __future__ import annotations

from openswe_traces.synth.bigl0 import BATCH_COVERAGE, CODEC_COVERAGE, unit_specs
from openswe_traces.synth.rules import RuleContext, check_b4


def test_coverage_tables_cover_contract_rows() -> None:
    assert len(CODEC_COVERAGE) >= 11
    assert len(BATCH_COVERAGE) >= 7
    codec_blob = " ".join(a for a, _ in CODEC_COVERAGE)
    assert "TestCodecUnmentionedRandom" in codec_blob
    assert "TestLocateMalformedRegionNoBackoff" in codec_blob
    batch_blob = " ".join(a for a, _ in BATCH_COVERAGE)
    assert "TestBatchUnmentionedRandom" in batch_blob
    assert "TestBatchStreamGroupingProperty" in batch_blob


def test_hidden_suites_are_black_box() -> None:
    from openswe_traces.data import ROOT

    author = ROOT / "experiments/harbor_nex/tasks_bigL0"
    units = unit_specs(author)
    for unit in units:
        ctx = RuleContext(task_dir=author / unit.family)
        ctx.hidden_files = {h.relpath: h.content for h in unit.hidden}
        verdict = check_b4(ctx)
        assert not verdict.skipped
        assert verdict.passed, verdict.evidence
        blob = "\n".join(h.content for h in unit.hidden)
        for tok in ("ThornSlot", "codecV2", "*codecV2", "KelpBolt", "ThornPipe", "batchCommandsBuilder"):
            assert tok not in blob, tok
        assert "20260919" in blob
        assert "10000" in blob or "batchCases = 10000" in blob
