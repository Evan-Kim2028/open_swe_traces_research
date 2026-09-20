# Difficulty — certdesc

predicted_flip: L3
details: 6
Commitments: OID→short-name mapping with numeric fallback, Go-identifier usage naming, exact-match parse pairs with ok-returns, `ExtKeyUsage:<n>` unknown fallback, sorted+joined description with a canonical-combo collapse table, and shared comma-format symmetry with IssueCert.

Hardness driver: three parallel string tables each with a different fallback behavior (numeric OID / bit-name / `ExtKeyUsage:<n>`), plus the sorted-join-then-collapse indirection where the canonical strings ARE the test oracle.
