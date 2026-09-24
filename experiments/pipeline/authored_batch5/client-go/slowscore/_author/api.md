# Exported API — slowscore

Package `internal/locate` (module `example.internal/kvstore/v2`) — store
health scoring used by replica selection.

Unexported but in-package testable: `CountSlidingWindow` (`Append(value)
gradient`, `Avg`, `Sum`), `SlowScoreStat` (`recordSlowScoreStat(duration)`,
`updateSlowScore()`, `getSlowScore`, `isSlow`, `markAlreadySlow`,
`resetSlowScore`), `replicaFlowsType.String`.

Callers: `region_cache.go` (`checkAndUpdateStoreSlowScores` tick,
`filterStoreCandidate` deprioritizes slow stores), `region_request.go`
liveness checks. In-tree tests: `TestSlowScoreStat` removed from
`region_cache_test.go`; other suites transitively exercise the stubs.
