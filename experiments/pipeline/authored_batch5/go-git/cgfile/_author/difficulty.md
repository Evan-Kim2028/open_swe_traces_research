# Difficulty — cgfile

predicted_flip: L2
details: 14

Missed edges: per-failure error split; size precheck skip on unsizable
readers; TOC monotonicity + duplicate-id + early-terminator guards; chunk
sizes from adjacent offsets; cardinality checks at open; parent-chain
delegation below the minimum; octopus edge-walk bounds; generation
overflow indirection; dropped unterminated last chain line; open-vs-parse
fallback split; parent close error suppression.

Hardness driver: a binary container whose guards all encode upstream
audit rules — the kept comments cite them, but the enforcement points are
what's graded.
