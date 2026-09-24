# Closure — slowscore

Package: `internal/locate`. File: `internal/locate/slow_score.go` (186 lines).

Removed (all bodies stubbed): `CountSlidingWindow.Avg`, `.Sum`, `.Append`;
`SlowScoreStat.getSlowScore`, `.updateSlowScore`, `.recordSlowScoreStat`,
`.markAlreadySlow`, `.resetSlowScore`, `.isSlow`; `replicaFlowsType.String`.

Kept: the `slowScore*`/`slidingWindowSize` constants, both struct
definitions, the `replicaFlowsType` enum.

Tests edited: `TestSlowScoreStat` removed from `region_cache_test.go`
(24 lines; sole direct coverage of the closure). Other locate tests stay —
they hit the stubs transitively through replica selection, which is the
intended failure signal.
