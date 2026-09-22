# Contract (L2) — slowscore

`CountSlidingWindow` keeps at most `slidingWindowSize` samples; below the
cap `Append` grows the sum, at cap it evicts the oldest sample first, and
`Avg` is integer `sum/len`. `Append` returns a gradient — the relative
change from the previous average when that average is positive and the
value differs — and a small positive floor otherwise. `updateSlowScore`
on an uninitialized stat only seeds `avgScore` and `avgTimecost` and
returns without a window update. On a tick with recorded updates the two
gradients come from the update-count and average-timecost sliding
windows; with zero updates both default to `1.0` and only the decay path
can run. The score rises only when the update gradient falls while the
latency gradient rises; otherwise it decays toward 1 and snaps to 1 when
within one decay step of it. Each tick ends by copying the timecost
window's average into `avgTimecost` and zeroing both interval counters.
`recordSlowScoreStat` counts the request and accumulates its
microseconds, but on an uninitialized stat it only seeds and never
reaches the max-timeout check; once initialized a single request at or
above the max-timeout bound pins the score to its maximum immediately.
`isSlow` is the committed threshold comparison; `markAlreadySlow` pins
the maximum score; `resetSlowScore` clears it.
`replicaFlowsType.String` renders the named flow kinds and falls back to
a decimal rendering for anything else.

## Coverage

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | the sliding window caps at `slidingWindowSize`, evicts oldest first, and `Avg` is integer division |
| `TestDetail02` | `Append` returns a relative-change gradient with a positive floor when the average is non-positive or unchanged |
| `TestDetail03` | `updateSlowScore` on an uninitialized stat only seeds and returns |
| `TestDetail04` | gradients come from the two windows; zero updates default both to `1.0` and only decay runs |
| `TestDetail05` | the score rises only when updates fall while latency rises |
| `TestDetail06` | otherwise the score decays toward 1 and snaps within one step |
| `TestDetail07` | each tick ends by copying `avgTimecost` and zeroing the interval counters |
| `TestDetail08` | `recordSlowScoreStat` seeds-only when uninitialized; a max-timeout request pins the score once initialized |
| `TestDetail09` | `isSlow`/`markAlreadySlow`/`resetSlowScore` implement the threshold, pin, and clear |
| `TestDetail10` | `replicaFlowsType.String` renders named kinds and a decimal fallback |
