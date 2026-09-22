# Difficulty — clusterops

predicted_flip: L3
details: 8

Missed edges: ConfVer bumps on peer add/remove and Version bumps on
range/merge changes; leader zeroing when the removed peer led; the
positional peerIDs↔storeIDs panic contract; `UpdateStoreAddr` clobbering
PeerAddress; evacuate-range splitting a covering region into two
(count=3 → 2 regions on empty data); quotient+remainder distribution.

Hardness driver: 36 functions is a wide surface, and a plausible impl
that gets the shapes right but skips `RegionEpoch` bookkeeping passes
every structural example while failing the epoch assertions.
