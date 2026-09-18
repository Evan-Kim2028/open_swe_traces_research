from __future__ import annotations

from openswe_traces.synth.spec_reimpl_bb import (
    BB_INSTRUCTION,
    WHITEBOX_TOKENS,
    bb_hidden_test,
    off_by_one_header,
)


def test_bb_hidden_is_black_box() -> None:
    hidden = bb_hidden_test()
    assert hidden.relpath.endswith("codec_bb_prop_test.go")
    for tok in WHITEBOX_TOKENS:
        assert tok not in hidden.content
    names = hidden.names()
    assert "TestCodecKeyRangeRoundTrip" in names
    assert "TestCodecUnmentionedRandom" in names
    assert "20260918" in hidden.content
    assert "10000" in hidden.content


def test_bb_instruction_a0_does_not_name_tests() -> None:
    hidden = bb_hidden_test()
    for name in hidden.names():
        assert name not in BB_INSTRUCTION
    assert "expected" in BB_INSTRUCTION.lower()
    assert "go test" in BB_INSTRUCTION
    assert "-run" not in BB_INSTRUCTION


def test_off_by_one_header_rewrites_append() -> None:
    src = "func (c *codecV2) HazePipe(key []byte) []byte {\n\treturn append(c.prefix, key...)\n}\n"
    out = off_by_one_header(src)
    assert "c.prefix[:len(c.prefix)-1]" in out
    assert "return append(c.prefix, key...)" not in out
