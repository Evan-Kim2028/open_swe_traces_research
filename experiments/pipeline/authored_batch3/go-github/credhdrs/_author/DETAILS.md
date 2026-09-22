# DETAILS — credhdrs

1. The returned request is a copy — the caller's request is never mutated
   (http.RoundTripper contract). Inferable: yes — the clone-then-return shape
   is kept in the stub and the RoundTripper contract is documented upstream.
2. The copy carries a Basic-Auth Authorization header built from the id/secret
   arguments. Inferable: yes — the func name and transport callers state it.
3. The copied request's Header map is independent of the original's — mutating
   it must not leak back. Inferable: partially — Clone() already provides it;
   the requirement that auth lands on the copy (not the original) is the
   point of the bug.
4. Transports consult an origin-scope check before attaching credentials —
   credentials only flow to requests targeting the client's own base-URL
   origin. Inferable: partially — scoping is visible at the call sites, the
   exact origin rule lives in a sibling unit's closure (urlpolicy).
