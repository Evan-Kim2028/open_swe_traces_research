# Contract (L2) — coalesce

User-supplied values are merged over chart defaults to produce the render values. For each chart (and recursively each of its dependencies), the user's table overrides the chart's default table key-by-key: a user key missing from defaults is added; a user value of nil deletes the default key; mismatched types produce a warning and the user value wins; nested tables merge recursively. In merge mode, tables are merged wholesale rather than coalesced per-key. Dependencies are processed under each of their names — both the chart name and every alias — and a subchart's own defaults fill gaps the user left. `global:` tables are special: the top-level global table is propagated into every subchart's table (without overwriting keys the subchart already has, and without a subchart's own nil shadowing the global), and dependency-level globals merge likewise. Nil values inside empty maps are cleaned so a user nil erases a subchart default but a subchart nil does not shadow a global. The result contains no keys the user set to null, and chart dependencies' value tables appear nested under their names. Tables merge in place; the source is not mutated.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestCoalesceValues` | coalesce user over defaults |
| `TestMergeValues` | merge mode wholesale |
| `TestCoalesceTables` | table-level coalesce |
| `TestMergeTables` | table-level merge |
| `TestCoalesceValuesWarnings` | type-mismatch warnings |
| `TestCoalesceValuesEmptyMapWithNils` | nil cleanup in empty maps |
| `TestCoalesceValuesSubchartDefaultNilsCleaned` | subchart nil defaults cleaned |
| `TestCoalesceValuesUserNullErasesSubchartDefault` | user null erases default |
| `TestCoalesceValuesSubchartNilDoesNotShadowGlobal` | subchart nil vs global |
| `TestCoalesceValuesSubchartNilCleanedWhenUserPartiallyOverrides` | partial override nil cleanup |
