# Contract (L2) — depresolver

Dependency resolution turns a chart's declared requirements plus the repo-name→URL mapping into a lock file listing concrete resolved dependencies with a content hash. Each requirement's repository field selects a resolution path: an OCI (`oci://`) reference resolves through the registry client; `file://` (absolute or relative to the chart directory) resolves to a local path, which must exist, be a chart directory or archive, and is digested from disk — its version is read from the chart itself rather than the constraint; `alias:` names resolve under the aliased chart while preserving the alias name; `name@url` style repo names are looked up in the repo-names map. Every non-local dependency's requested version must satisfy its semver constraint, and resolution picks the version available in the index. A dependency missing from the repo map, an unsatisfiable constraint, or a missing local path is an error. The lock records dependencies in a stable order with their resolved repository; the lock hash is a sha256 over the canonicalized (sorted) requirement set, so two equivalent requirement lists hash identically and a different set hashes differently.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestResolve` | resolution across oci/file/repo/alias paths |
| `TestHashReq` | lock hash stability and difference |
| `TestGetLocalPath` | local path resolution and errors |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./internal/resolver/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
