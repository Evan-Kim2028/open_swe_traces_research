# Difficulty — mvccread

predicted_flip: L2
details: 6

Missed edges: the MaxUint64+primary-key short-circuit returns
`startTS - 1` (writer reading its own lock); pessimistic/lock ops never
block; rollback and lock-type values are skipped during version selection;
`lockErr` encodes the key at `lockVer`.

Hardness driver: the guard chain has three early exits whose order and
conditions are easy to swap; a check missing the MaxUint64 branch passes
every ordinary read.
