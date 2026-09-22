# Exported API — deltasel

Package `plumbing/format/packfile` — `DeltaSelector`, the sliding-window
policy that decides which objects become deltas and against which base.

`NewDeltaSelector(storer.EncodedObjectStorer)`, `ObjectsToPack(hashes,
packWindow)`; internal: `objectsToPack`, `encodedDeltaObject`,
`encodedObject`, `fixAndBreakChains[One]`, `restoreOriginal`, `undeltify`,
`sort`, `walk`, `tryToDeltify`, `deltaSizeLimit`, `byTypeAndSize` sort impl;
consts `maxDepth=50`, `applyDelta` type map.

Caller: `Encoder.Encode`. In-tree tests removed: 4 (selector, cycle,
encoder suites); delta/patch/scanner suites stay.
