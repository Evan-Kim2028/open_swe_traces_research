# Difficulty — tfhcl2

predicted_flip: L2
details: 14
Commitments: sorted+run-aligned object bodies, kind→element mapping incl. nil-omission, empty-slice omission, slice-kind-dependent collapse, cty-tag/snake_case field keys with acronym edge cases, quoted-key map bodies with suppressed empties, minimal escape set, section ordering, locals-vs-output split, per-provider body rules, sorted type/name nesting, and provider version pins with fatal fallback.

Hardness driver: the output is byte-exact — alignment runs, suppressed empties, acronym snake_casing, and five per-provider body rules all have to land; every one of them is a plausible-wrong-choice commitment.
