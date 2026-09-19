# Bug report

The discovery HTTP service panics on startup or returns empty/403/404 for every universe. Endpoints registered in one universe leak into another (or vanish). The well-known OIDC document and JWKS merge (same key id, newer LastSeen wins) no longer work, and apply rejects valid client-cert identities.

Reproduce with:

```
go test -count=1 ./discovery/pkg/discovery/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
