# Difficulty — execfmt

predicted_flip: L2
details: 6

Missed edges: inclusive `<=1µs` boundary; decimal rounding that carries
(`5.999s`→`6s`); V1 `TotalRpcWallTimeNs` in NANOSECONDS amid `*Ms`
fields and no SuspendTime on the V1 path; V2-preferred precedence;
ScanDetailV2 field renames; nil-receiver nil-arg Update.

Hardness driver: merge-accumulation looks trivial, so a plausible impl
writes uniform `+=` loops and misses the unit asymmetries and the V2
preference.
