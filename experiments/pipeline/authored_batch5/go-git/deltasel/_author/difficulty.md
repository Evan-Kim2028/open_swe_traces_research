# Difficulty — deltasel

predicted_flip: L3
details: 14

Missed edges: window-0 passthrough; descending type/size sort; contiguous
group split; first-error-wins under concurrency; blob/tree-only filter;
1/16 size floor; depth-scaled limit + 8-byte floor + depth-cap zero;
delta reuse and original cleanup; absent-base and non-DeltaObject
undeltify; cycle break on visiting set; window-edge index eviction;
same-type break in backward scan; depth reset on undeltify.

Hardness driver: every guard is a magic constant or an ordering
convention — the observable behavior (pack size, hang, undecodable pack)
never reveals which constant was wrong.
