# Closure — clientclone

Package: github (root). File: github/github.go.
Removed bodies: Client.Clone — stubbed to `o := clientOptions{}` plus
the caller's opts, then `newClient(o)`: drops the `errUninitialized`
guard on a nil inner client, the field carry-over (API version range,
user agent, base/upload URLs, auth token, rate-check flags, retry
bound, marketplace stub), the transport choice (fresh base transport
when the clone carries a token so its auth wrapper scopes to its own
origins; shared wrapped transport otherwise) with CheckRedirect/Jar/
Timeout copy, and the shared `rateLimits` map hand-off under `rateMu`.
Kept: clientOptions, newClient, Client fields, WithX option funcs.
Tests removed: 4 funcs across github/github_test.go and
github/dependency_graph_test.go (TestClient_Clone,
TestClient_CloneReScopesToken, TestClientCopy_leak_transport,
TestDependencyGraphService_FetchSBOM_Download).
