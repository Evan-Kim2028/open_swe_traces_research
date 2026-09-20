# Details — fieldmap

1. `HumanPath*` returns the v1alpha2 spelling: it looks the path up in the mapping table's v1alpha3 column and returns the v1alpha2 column. Inferable: partially — direction is the commitment.
2. `InternalPath*` returns the v1alpha3 spelling: v1alpha2 input maps to v1alpha3 output. Inferable: partially.
3. A path with no mapping entry passes through verbatim in both directions. Inferable: partially — passthrough-vs-error is a choice.
4. Only the five table entries translate; matching is exact-string, first match wins. Inferable: yes — the table is visible.
5. `NewClusterField` stores the path verbatim; the exported wrappers delegate `NewClusterField(path).XxxPath()`. Inferable: yes.
6. `HumanPath` is named for display but returns the OLD (v1alpha2) spelling — the direction is backwards from the name. Inferable: no — counterintuitive assignment.
