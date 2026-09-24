# Difficulty — slowscore

predicted_flip: L2
details: 10

Missed edges: first `recordSlowScoreStat` only seeds and skips the >=30s pin;
gradients default to 1.0 on an idle tick; the rise rule needs falling
throughput AND rising latency simultaneously; decay snaps to 1 inside a
band; `1e-6` gradient floor vs the `0.1` thresholds.

Hardness driver: the update is a magic-constant EWMA whose guard
inequalities are easy to flip, and the initialization path is a hidden edge.
