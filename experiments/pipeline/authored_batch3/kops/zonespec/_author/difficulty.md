# Difficulty — zonespec

predicted_flip: L2
details: 10
Commitments: three spec forms (name / `*/id` / `name`+`id`), first-slash-only split, wildcard sentinels, empty-list-means-permit-all, dot-suffix normalization on both parse and match sides, and the asymmetric name-skip vs id-veto in matching.

Hardness driver: the `continue` vs `return false` asymmetry between name and id mismatches is invisible and easy to invert; `*/` and `*/*` sentinel handling overlaps in a way that invites a too-broad wildcard check.
