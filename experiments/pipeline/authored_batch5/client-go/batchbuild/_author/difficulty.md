# Difficulty — batchbuild

predicted_flip: L2
details: 6

Missed edges: `>=10` priority exemption from the limit AND the
drain-at-limit-0 rule; forwardedHost routing into a per-host map while
still consuming idAlloc; `reset()` preserving live queued entries and
never touching `idAlloc`; canceled entries skipping id assignment;
`clean` vs `reset` queue semantics.

Hardness driver: the plausible impl is a flat drain — count every
entry, put every request in one batch — which passes the normal-path
examples and fails every scheduling edge.
