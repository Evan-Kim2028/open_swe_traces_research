from __future__ import annotations

from pathlib import Path

from openswe_traces.synth.reconcile import (
    contract_grounding_offenders,
    gold_denylist,
    ground_examples,
    is_cold_repo,
    is_encoding_shape,
    parse_details,
    parse_hidden_funcs,
    prior_bank_unit_count,
    reconcile_unit,
    render_contract,
    rewrite_directional_ordering,
    scrub_b7,
    splice_contract,
    symbol_noun_phrase,
)

DETAILS = """# DETAILS — refescape

1. Refs are split on `/` and rejoined with literal `/` — separators are never encoded. Covered
   by `TestGitService_GetRef_pathEscape` — removed.
2. Each segment is percent-encoded independently: `#` → `%23`. Covered by UpdateRef — removed.
"""

HIDDEN = """package github_test

// Detail 1: refs split on '/' and rejoin with literal '/' — separators never encoded.
func TestDetail01_SlashSeparatorsPreserved(t *testing.T) {
	if strings.Contains(got, "%2F") {
		t.Fatalf("separator escaped: %q", got)
	}
	_ = "refs/heads/main"
}

// Detail 2: each segment percent-encoded independently.
func TestDetail02_PerSegmentEscaping(t *testing.T) {
	_ = "refs/heads/b#1"
	t.Fatalf("ref %q: request URI %q, want %q", ref, got, want)
}
"""

GOLD = """diff --git a/github/git_refs.go b/github/git_refs.go
--- a/github/git_refs.go
+++ b/github/git_refs.go
@@ -1,3 +1,8 @@
 func refURLEscape(ref string) string {
-	return ref // excised: refURLEscape
+	parts := strings.Split(ref, "/")
+	for i, s := range parts {
+		parts[i] = url.PathEscape(s)
+	}
+	return strings.Join(parts, "/")
 }
"""


def test_parse_details_drops_covered_by() -> None:
    ds = parse_details(DETAILS)
    assert [d.index for d in ds] == [1, 2]
    assert "Covered by" not in ds[0].text
    assert "separators are never encoded" in ds[0].text


def test_parse_hidden_pairs_detail_index() -> None:
    funcs = parse_hidden_funcs({"x_test.go": HIDDEN})
    assert [f.name for f in funcs] == [
        "TestDetail01_SlashSeparatorsPreserved",
        "TestDetail02_PerSegmentEscaping",
    ]
    assert funcs[0].index == 1
    assert funcs[1].index == 2
    assert "separators never encoded" in funcs[0].comment


def test_one_row_per_hidden_test_deterministic() -> None:
    result = reconcile_unit(
        unit="refescape",
        details_text=DETAILS,
        hidden_files={"x_test.go": HIDDEN},
        gold=GOLD,
        llm=False,
    )
    assert len(result.rows) == 2
    assert {r.test for r in result.rows} == {
        "TestDetail01_SlashSeparatorsPreserved",
        "TestDetail02_PerSegmentEscaping",
    }
    assert "| `TestDetail01_SlashSeparatorsPreserved` |" in result.contract
    assert "| `TestDetail02_PerSegmentEscaping` |" in result.contract
    # B7: gold symbols/files must not survive.
    assert "refURLEscape" not in result.contract
    assert "git_refs.go" not in result.contract


def test_llm_dropped_row_is_filled(tmp_path: Path) -> None:
    def fake(_prompt: str) -> dict:
        return {
            "invariants": "- slash separators stay literal.",
            "rows": [
                {
                    "test": "TestDetail01_SlashSeparatorsPreserved",
                    "sentence": "A slash-separated ref keeps `/` in the path.",
                }
            ],
            "examples": ["`refs/heads/main` keeps its slashes."],
            "unreconcilable": False,
            "unreconcilable_reason": "",
        }

    result = reconcile_unit(
        unit="refescape",
        details_text=DETAILS,
        hidden_files={"x_test.go": HIDDEN},
        gold=GOLD,
        llm=True,
        llm_complete=fake,
        cache_dir=tmp_path,
    )
    assert len(result.rows) == 2
    assert result.rows[1].test == "TestDetail02_PerSegmentEscaping"


def test_encoding_shape_flag() -> None:
    details = parse_details(
        "1. Flag false → plain omitempty marshal: nil parent fields are absent, not null.\n"
        "2. Flag true → parent_team_id emitted as null when unset.\n"
    )
    funcs = parse_hidden_funcs(
        {
            "t.go": "func TestDetail01_FlagOffOmitempty(t *testing.T) {}\n"
            "func TestDetail02_FlagOnParentsNullWhenUnset(t *testing.T) {}\n"
        }
    )
    assert is_encoding_shape(details, funcs)


def test_splice_keeps_bugreport() -> None:
    orig = (
        "# Contract (L2) — x\n\nold row\n\n# Bug report — x\n\n"
        "Expected: a. Got: b.\n\nReproduce with:\n\n```\ntests/test.sh\n```\n"
    )
    new = "# Contract (L2) — x\n\n- new invariant\n"
    out = splice_contract(orig, new)
    assert "new invariant" in out
    assert "old row" not in out
    assert "# Bug report — x" in out
    assert "Reproduce with:" in out


def test_scrub_b7_strips_gold_symbols() -> None:
    deny = gold_denylist(GOLD)
    assert "refURLEscape" in deny
    assert "git_refs.go" in deny
    cleaned, leaks = scrub_b7("call refURLEscape in git_refs.go:12", deny)
    assert "refURLEscape" not in cleaned
    assert leaks


def test_render_one_row_per_test() -> None:
    text = render_contract(
        "u",
        "- a",
        [
            type("R", (), {"test": "TestDetail01_A", "sentence": "does a"})(),
            type("R", (), {"test": "TestDetail02_B", "sentence": "does b"})(),
        ],
    )
    assert text.count("| `TestDetail") == 2


def test_cold_repo_counts_bank_not_batch2(tmp_path: Path) -> None:
    bank = tmp_path / "experiments" / "pipeline" / "tasks_composerver" / "helm"
    (bank / "chartloader-L2").mkdir(parents=True)
    (bank / "chartloader-L0").mkdir()
    (bank / "coalesce-L2").mkdir()
    (bank / "depresolver-L2").mkdir()
    (bank / "_skel").mkdir()
    assert prior_bank_unit_count("helm", root=tmp_path) == 3
    assert not is_cold_repo("helm", root=tmp_path)
    assert prior_bank_unit_count("go-github", root=tmp_path) == 0
    assert is_cold_repo("go-github", root=tmp_path)


def test_does_not_need_bugreport() -> None:
    # Reconciler takes details/hidden/gold only — bugreport is not an input.
    result = reconcile_unit(
        unit="refescape",
        details_text=DETAILS,
        hidden_files={"x_test.go": HIDDEN},
        gold=GOLD,
        llm=False,
    )
    assert "Bug report" not in result.contract


# ---------------------------------------------------------------------------
# reconciler-v2 fixes — fixtures are the EXACT failing text from the
# helm-repindex generated contract (sweep_rc_auto, scored 0/3 at L2).
# ---------------------------------------------------------------------------

REPINDEX_HIDDEN = (
    Path(__file__).resolve().parents[1]
    / "src"
    / "openswe_traces"
    / "synth"
    / "testdata"
    / "composerver"
    / "helm"
    / "repindex_bb_prop_test.go"
).read_text(encoding="utf-8")

# The direction-stated ordering claim that produced `beta not sorted desc`
# on 2 of 3 solver attempts.
FAILING_ORDERING_SENTENCE = (
    "SortEntries orders each chart's versions in descending semantic version "
    "order (prereleases sort before releases of same core version)."
)

# The invented worked example: a range constraint does not match a
# pre-release, and no assertion in the suite supports it.
FAILING_UNGROUNDED_EXAMPLE = (
    'Get(name="app", version=">=1.0.0") with versions '
    '["0.9.0", "1.0.0", "1.2.0", "2.0.0-alpha"] → returns "2.0.0-alpha" '
    "(highest matching)"
)

GOLD_REPINDEX = """diff --git a/pkg/repo/v1/index.go b/pkg/repo/v1/index.go
--- a/pkg/repo/v1/index.go
+++ b/pkg/repo/v1/index.go
@@ -1,3 +1,8 @@
+func (i *IndexFile) MustAdd(md *chart.Metadata, name, base, digest string) error {
+}
+func (i *IndexFile) SortEntries() {
+}
+func (i *IndexFile) Merge(o *IndexFile) {
+}
+func (i *IndexFile) Has(name, version string) bool {
+}
+func IndexDirectory(dir, baseURL string) (*IndexFile, error) {
+}
+func LoadIndexFile(path string) (*IndexFile, error) {
+}
"""


def test_ordering_never_stated_as_direction() -> None:
    out, changed = rewrite_directional_ordering(
        FAILING_ORDERING_SENTENCE, REPINDEX_HIDDEN
    )
    assert changed
    # the class direction is gone
    assert "prereleases sort before releases" not in out
    assert "sort before" not in out
    # pairwise rule: specific values, in descending position
    assert "comes before" in out
    assert "`0.1.0-beta`" in out  # suite literal (>=0.1.0-beta)
    assert "`0.0.1`" in out  # suite literal (!=0.0.1)
    # build metadata's participation is stated
    assert "build metadata" in out


def test_ordering_pairwise_claim_untouched() -> None:
    pairwise = (
        "after sorting, `0.0.3-beta.2` comes before `0.0.1` and `1.0.0` "
        "comes before `1.0.0-beta`."
    )
    out, changed = rewrite_directional_ordering(pairwise, REPINDEX_HIDDEN)
    assert not changed
    assert out == pairwise


def test_worked_examples_must_be_grounded() -> None:
    kept, dropped = ground_examples(
        [
            FAILING_UNGROUNDED_EXAMPLE,
            'Get("solo", "") with versions ["1.0.0"] → returns "1.0.0"',
        ],
        REPINDEX_HIDDEN,
    )
    assert kept == ['Get("solo", "") with versions ["1.0.0"] → returns "1.0.0"']
    assert dropped == [FAILING_UNGROUNDED_EXAMPLE]


def test_selfcheck_fails_unit_on_surviving_ungrounded_example(tmp_path: Path) -> None:
    def fake(_prompt: str) -> dict:
        return {
            "invariants": "- index lookup resolves queries.",
            "rows": [
                {
                    "test": "TestRIGet",
                    # invented output literal, not excisable as a clause:
                    # the self-check must fail the unit.
                    "sentence": 'Get(">=1.0.0") returns "2.0.0-alpha" '
                    "(highest matching).",
                }
            ],
            "examples": [],
            "unreconcilable": False,
            "unreconcilable_reason": "",
        }

    result = reconcile_unit(
        unit="repindex",
        details_text="1. lookup resolves version queries.\n",
        hidden_files={"repindex_bb_prop_test.go": REPINDEX_HIDDEN},
        gold=GOLD_REPINDEX,
        llm=True,
        llm_complete=fake,
        cache_dir=tmp_path,
    )
    assert result.unreconcilable
    assert "ungrounded" in result.unreconcilable_reason
    assert "2.0.0-alpha" not in result.contract or "grounding" in result.unreconcilable_reason


def test_scrub_b7_descriptive_noun_phrases() -> None:
    deny = gold_denylist(GOLD_REPINDEX)
    cleaned, _ = scrub_b7(
        "SortEntries orders each chart's versions; Merge combines two indexes; "
        "IndexDirectory walks a directory for *.tgz files; "
        "Has(name, version) is true exactly when Get(name, version) succeeds.",
        deny,
    )
    assert "the call" not in cleaned
    assert "sorting the entries" in cleaned
    assert "merging combines two indexes" in cleaned
    assert "the directory scan walks a directory" in cleaned
    assert "presence-testing(name, version)" in cleaned
    # B7 still forbids the symbols themselves
    for sym in ("SortEntries", "Merge", "IndexDirectory", "Has"):
        assert sym not in cleaned


def test_symbol_noun_phrase_mapping() -> None:
    assert symbol_noun_phrase("Merge") == "merging"
    assert symbol_noun_phrase("IndexDirectory") == "the directory scan"
    assert symbol_noun_phrase("Has") == "presence-testing"
    assert symbol_noun_phrase("MustAdd") == "adding"
    assert symbol_noun_phrase("LoadIndexFile") == "loading the index file"


def test_repindex_failing_contract_through_pipeline(tmp_path: Path) -> None:
    """The whole v1 contract failure, replayed: direction-ordering claim +
    invented example + 'the call' substitutions must all be gone."""

    def fake(_prompt: str) -> dict:
        return {
            "invariants": (
                "- SortEntries orders each chart's versions in descending "
                "semantic version order (prereleases sort before releases of "
                "same core version).\n"
                "- Merge combines two indexes; deduplication key is (name, "
                "semver-equal version ignoring build metadata)."
            ),
            "rows": [
                {
                    "test": "TestRIAddSort",
                    "sentence": FAILING_ORDERING_SENTENCE,
                },
                {
                    "test": "TestRIGet",
                    "sentence": "Get(name, version) resolves a range to the "
                    "highest matching version; Has(name, version) is true "
                    "exactly when Get(name, version) succeeds.",
                },
                {
                    "test": "TestRIMerge",
                    "sentence": "Merge combines two indexes.",
                },
                {
                    "test": "TestRIIndexDirectory",
                    "sentence": "IndexDirectory walks a directory for *.tgz "
                    "files.",
                },
            ],
            "examples": [
                FAILING_UNGROUNDED_EXAMPLE,
                'Get("solo", "") with versions ["1.0.0"] → returns "1.0.0"',
            ],
            "unreconcilable": False,
            "unreconcilable_reason": "",
        }

    result = reconcile_unit(
        unit="repindex",
        details_text="1. index lookup resolves version queries.\n",
        hidden_files={"repindex_bb_prop_test.go": REPINDEX_HIDDEN},
        gold=GOLD_REPINDEX,
        llm=True,
        llm_complete=fake,
        cache_dir=tmp_path,
    )
    c = result.contract
    assert "the call" not in c
    assert "prereleases sort before releases" not in c
    assert "2.0.0-alpha" not in c
    assert "comes before" in c
    assert "merging combines two indexes" in c
    assert "the directory scan walks a directory" in c
    assert "presence-testing(name, version)" in c
    assert not result.unreconcilable
    assert any("pairwise" in f for f in result.applied_fixes)
    assert any("ungrounded" in f for f in result.applied_fixes)


def test_grounding_offenders_flags_marked_fragments() -> None:
    offenders = contract_grounding_offenders(
        "rows return errors.\n- e.g., Get(\"x\") → \"9.9.9-unseen\"\n",
        REPINDEX_HIDDEN,
    )
    assert offenders
