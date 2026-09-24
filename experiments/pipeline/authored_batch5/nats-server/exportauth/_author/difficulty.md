# difficulty — exportauth

Medium. A predicate chain, not a parser: the traps are order (public
short-circuit before accountPos before tokenReq before approved-map),
direction (export pattern must cover the requesting subject, tested via
isSubsetMatch on the import tokens), 1-based accountPos indexing into the
requester's tokens, and revocation freshness (t < issuedAt means stale,
plus the jwt.All fallback). Map iteration order makes "first match wins"
observable only through subset semantics.
