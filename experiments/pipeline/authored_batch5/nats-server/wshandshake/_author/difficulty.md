# Difficulty — wshandshake

predicted_flip: L2
details: 8

Missed edges: PMC param scan is scoped to the extension's own params and
requires BOTH no-context-takeover params; checkPMCOnly short-circuit;
missing-port AddrError → scheme default (443/80) + host lowercasing;
absent Origin accepts; sameOrigin checks host+port+scheme where the
request's scheme comes from r.TLS; explicit-port same-origin mismatch;
allowed list keyed by host but matching (scheme,port); FIPS carve-out for
SHA-1.

Hardness driver: header-token parsing plus an origin policy whose accept
paths are deliberately permissive — easy to make too strict or too lax.
