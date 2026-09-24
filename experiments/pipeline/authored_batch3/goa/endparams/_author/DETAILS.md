# Commitments — endparams

1. PathParams returns the subset of endpoint Params named by route
   wildcards, carrying requiredness; QueryParams returns the remaining
   params, splitting "attribute:name" keys on ":" before checking
   required. In-tree coverage: deleted tests only. Inferable:
   partially.
2. Params collects wildcard names across all full paths without
   duplicates. In-tree coverage: deleted tests only. Inferable: yes.
3. FullPaths joins each service base path with the route path and
   preserves a trailing slash when the route path ends in one (or the
   route is "/" over a base ending in "/"); absolute routes ("//" prefix)
   skip the base join. In-tree coverage: deleted tests only.
   Inferable: no — the trailing-slash rule is a subtle detail.
4. IsAbsolute is the "//" path prefix. In-tree coverage: none.
   Inferable: doc.
5. validateParams rejects object/map/union path params (map allowed for
   query), non-primitive array elements, params missing from the
   payload, and >1 param on array/map payloads. In-tree coverage:
   deleted tests only. Inferable: no.
6. validateHeadersAndCookies rejects object/union headers and any
   non-primitive cookie (arrays rejected), non-primitive array
   elements, mappings missing from payload, an "Authorization" header
   mapping under basic auth, and any header on a map payload. In-tree
   coverage: deleted tests only. Inferable: no.
7. RouteExpr.Validate reports params missing from payload, a map or
   multi-param primitive payload, duplicate wildcards in a full path,
   non-GET WebSocket routes, and HEAD responses with bodies. In-tree
   coverage: deleted tests only. Inferable: no — the HEAD/WS checks are
   RFC-derived policy.
