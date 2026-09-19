# Exported API — oidcdisc

`NewMemoryStore() *MemoryStore` implements `Store`.

`NewServer(store Store) *Server` — `http.Handler` via `ServeHTTP`.

Public (no client cert): `GET /{universe}/.well-known/openid-configuration`, `GET /{universe}/openid/v1/jwks`.

Authenticated: list/create/get/patch discovery endpoints under `/{universe}/apis/discovery.clustkit.k8s.io/v1alpha1/...`.

Callers: the discovery service binary. In-tree tests build a TLS httptest server from `NewServer(NewMemoryStore())` and exercise isolation, OIDC documents, JWKS merge, and server-side apply.
