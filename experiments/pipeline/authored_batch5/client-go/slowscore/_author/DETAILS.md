# Details — slowscore

1. `CountSlidingWindow` keeps at most `slidingWindowSize` (10) samples; below
   cap `Append` grows `sum`, at cap it evicts the oldest sample first.
   `Avg` is `sum/len(history)` integer division. Inferable: partially —
   windowing is described, eviction order is not.
2. `Append` returns the gradient `(value-prevAvg)/prevAvg` when the previous
   average is positive and differs; otherwise `1e-6`. Inferable: no — the
   floor value is arbitrary.
3. `updateSlowScore` on an uninitialized stat (`avgTimecost == 0`) only seeds
   `avgScore = 1` and `avgTimecost = 500000` (µs) and returns — no window
   update. Inferable: no.
4. Per tick with `intervalUpdCount > 0`, gradients come from the two sliding
   windows (update-count gradient, interval-average-timecost gradient); with
   zero updates both gradients default to `1.0` and only the decay path can
   run. Inferable: no.
5. Score RISES only when `updGradient + 0.1 <= 0` AND `tsGradient >= 0.1`
   (throughput falling while latency rises); the new score is
   `ceil(min(score * min(5.43, |ts/upd|) + 1, 100))`. Inferable: no — magic
   constants.
6. Otherwise the score DECAYS toward 1 by `costScore = ceil(max(1, min(2.71,
   1+|updGradient|)))`; if the current score is within `costScore` of the
   init value it snaps to 1. Inferable: no.
7. Each tick ends by CAS-copying `avgTimecost` from the timecost window's
   average and zeroing both interval counters. Inferable: partially.
8. `recordSlowScoreStat` counts the request and accumulates its microseconds
   — but on an uninitialized stat it ONLY seeds (score=1, avgTimecost=500000,
   intervalTimecost=this request) and never reaches the max-timeout check.
   Once initialized, a single request of `>= 30000000`µs pins the score to
   100 immediately. Inferable: no.
9. `isSlow()` is `score >= 80`; `markAlreadySlow` pins 100;
   `resetSlowScore` stores 0. Inferable: doc for the threshold constant, the
   pin values are arbitrary.
10. `replicaFlowsType.String` renders `toLeader`/`toFollower` as
    `"ToLeader"`/`"ToFollower"` and anything else as its decimal. Inferable:
    partially — exact casing is a choice.
