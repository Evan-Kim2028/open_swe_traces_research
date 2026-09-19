# Difficulty — repindex

predicted_flip: L4
Two subtle semantics hide inside ordinary-looking code: semver-range resolution to the highest match (including the `isVersionRange` detection heuristic) and URL joining rules (absolute vs relative). Sort order and dedup-on-merge add a third. The contract describes them but a mid model usually implements exact-match only or joins URLs wrong; signatures at L4 plus test names are needed to land every branch.

Hardness driver: semver-range + URL-resolution invariants.
