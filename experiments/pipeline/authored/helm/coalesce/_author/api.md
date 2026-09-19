# Exported API — coalesce

`CoalesceValues(chrt, vals)` / `MergeValues(chrt, vals)` -> common.Values; `CoalesceTables(dst, src)` / `MergeTables(dst, src)` -> map[string]any.
Callers: engine rendering and ToRenderValues across action package.
