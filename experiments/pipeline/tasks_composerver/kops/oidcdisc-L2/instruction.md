# Contract (L2) — oidcdisc

The in-memory store partitions endpoints by universe id. Upsert creates the universe if needed, stamps Kind/APIVersion, sets LastSeen to now (RFC3339), and keys the object by namespace+name. List of an unknown universe is empty, not an error. Get of a missing object returns nil, nil.

HTTP: the handler mux is registered at construction. OIDC discovery requires a Host; issuer is `https://{host}/{universe}/` and jwks_uri is that issuer plus `openid/v1/jwks`. The document is 404 unless at least one endpoint in the universe has an OIDC spec; response_types_supported is `id_token`, subject_types `public`, algs `RS256`. JWKS unions keys by kid, skipping empty kids; when the same kid appears twice, the endpoint with the lexicographically greater LastSeen wins.

Write paths: create/apply decode JSON. The object name, if set, must equal the client-cert identity; namespace (and for apply, name) must match the URL or the handler returns 403. Successful create/apply return 201 with the body. List may be cluster-scoped or namespaced (filter by URL namespace). Get 404s when missing.

Universes do not leak: listing or OIDC in universe A never sees objects upserted in B.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestDiscoveryIsolation` | objects in one universe are invisible in another |
| `TestOIDCDiscovery` | well-known document issuer/jwks_uri and 404 when no OIDC spec |
| `TestOIDCMerging` | JWKS merge by kid with LastSeen winner |
| `TestServerSideApply` | apply/create identity and namespace checks; listed object is the applied one |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./discovery/pkg/discovery/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
