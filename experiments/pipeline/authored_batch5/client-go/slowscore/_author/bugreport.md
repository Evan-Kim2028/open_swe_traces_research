# Bug report

Store health scoring in `internal/locate` panics: sliding-window statistics,
the periodic score update, request recording, and the slow-store predicate
are all stubbed, so replica selection cannot deprioritize slow stores.

Expected (in-package probes): a fresh `CountSlidingWindow` appending 10
reports gradient `1e-6`, average `10`, sum `10`; after ten samples of 5,
appending 105 evicts the oldest (sum 50 → 150, average 15, gradient 20). A
fresh `SlowScoreStat` ticks to score 1. After initialization, a single 40s
request pins the score to 100 and `isSlow()` is true;
`markAlreadySlow`/`resetSlowScore` set 100/0. `replicaFlowsType(0).String()`
is `"ToLeader"`, `(1)` is `"ToFollower"`, `(7)` is `"7"`.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
