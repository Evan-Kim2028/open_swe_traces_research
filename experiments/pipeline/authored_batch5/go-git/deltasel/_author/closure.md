# Closure — deltasel

Package: `plumbing/format/packfile`. File: `delta_selector.go`.

Removed (16 functions stubbed): `NewDeltaSelector`,
`DeltaSelector.ObjectsToPack`, `.objectsToPack`, `.encodedDeltaObject`,
`.encodedObject`, `.fixAndBreakChains`, `.fixAndBreakChainsOne`,
`.restoreOriginal`, `.undeltify`, `.sort`, `.walk`, `.tryToDeltify`,
`.deltaSizeLimit`, `byTypeAndSize.Len/Swap/Less`.

Kept: `DeltaSelector` type + doc (incl. the pre-selection passthrough
idiom), `maxDepth` const with JGit citation, `applyDelta` type map,
`byTypeAndSize` type, `ObjectsToPack` doc comment (packWindow semantics);
`deltaIndex`/`getDelta`/`ObjectToPack` stay implemented — the delta
machinery is a readable sibling.

Tests deleted: `delta_selector_test.go`, `delta_selector_cycle_test.go`,
`encoder_test.go`, `encoder_advanced_test.go` (4 — the suites that pin
selection policy; encoder tests exercise it end-to-end).
