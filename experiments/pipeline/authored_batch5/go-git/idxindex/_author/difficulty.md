# Difficulty — idxindex

predicted_flip: L3
details: 14

Missed edges: bucketed binary search vs flat scan; MayContain fanout-only
answer; 64-bit overflow indirection; once-built reverse map; two distinct
iteration orders; prefix early-stop; lazy ReadAt section arithmetic; .rev
path; Writer pre-footer/count guards; dedupe-on-add; overflow table
ordering; close-time handle release.

Hardness driver: three implementations of one interface with identical
observable results — the differences (fd lifetime, laziness, ordering)
only surface under concurrency, big-offset and prefix probes.
