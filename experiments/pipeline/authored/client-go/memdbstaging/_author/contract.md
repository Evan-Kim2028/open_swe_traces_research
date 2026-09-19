# Contract (L2) — memdbstaging

Writes go into the current staging level; Staging starts a new level and returns a handle; Release(h) merges level h into its parent (writes persist, flags persist); Cleanup(h) drops everything written at level h (keys, values, and flag mutations revert). Nested levels compose: releasing the innermost then cleaning the outer removes both. RevertToCheckpoint restores the exact node set, value log position, and sizes captured by Checkpoint. Delete writes a tombstone (empty value) — Get returns not-found, Len still counts it until reset. Flags ops (SetWithFlags/DeleteWithFlags/UpdateFlags) attach or clear per-key flags; flagged keys are visible via InspectStage regardless of tombstone. Dirty reports uncheckpointed writes; Len/Size track live entries and arena bytes.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestGetSet` | set/get/delete round-trip including tombstones |
| `TestNestedSandbox` | nested staging levels merge/discard per contract |
| `TestInspectStage` | iterating a level yields only that level's entries with flags |
| `TestDirty` | dirty flag tracks writes vs checkpoint |
| `TestFlags` | flag set/clear/persist across staging levels |
| `TestReset` | reset empties the db |
| `TestDiscard` | discarded values unreadable |
| `TestFlushOverwrite` | overwrite in a later stage shadows earlier |
| `TestComplexUpdate` | mixed put/delete/flag ops across levels |
| `TestUnsetTemporaryFlag` | temporary flags cleared only by the right op |
| `TestMemDBStaging` | staging handles sequence correctly |
