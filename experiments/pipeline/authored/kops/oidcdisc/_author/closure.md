# Closure — oidcdisc

Package: discovery/pkg/discovery. Files: server.go, oidc.go, memory_store.go.

Removed: 12 functions (bodies stubbed to `panic("excised: <name>")`, signatures and doc comments preserved, compiles clean).

`NewServer`, `ServeHTTP`; list/create/apply/get discovery-endpoint handlers; OIDC discovery + JWKS handlers; `NewMemoryStore`, `UpsertDiscoveryEndpoint`, `ListDiscoveryEndpoints`, `GetDiscoveryEndpoint`.
