# Difficulty — igrole

predicted_flip: L3
details: 8
Commitments: case-insensitive role matching, unconditional `controlplane`→`control-plane` normalization, two-sided plural trim in lenient mode, the lenient-only `master` legacy alias, `("",false)` failure shape, strict unknown-field YAML, and empty-input no-op.

Hardness driver: lenient-vs-strict splits four behaviors that look like one flag (`controlplane` corrects even when strict); the `master` alias is a hidden compat path with no structural hint; the YAML empty-input no-op contradicts the strictness of the same function.
