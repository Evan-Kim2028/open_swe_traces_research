# Difficulty — clusterq

predicted_flip: L2
details: 8

Missed edges: `context.Canceled` when an unrelated store is canceled;
non-nil empty `Peer{}` for leaderless ScanRegions entries; deep-clone
aliasing on every getter; `GetPrevRegionByKey` panic on an unmatched key;
EndKey==startKey exclusion boundary; down-peer join scoped to the
returned region only.

Hardness driver: the plausible impl is a straightforward map scan that
returns live pointers — it passes every lookup example and silently
fails aliasing, the global-cancel check, and the leaderless-leader shape.
