# Closure — depresolver

Package: internal/resolver. Files: resolver.go.

Removed: 4 functions (bodies stubbed to `panic("excised: <name>")`, signatures and doc comments preserved, compiles clean).

`New(chartpath, cachepath, registryClient) *Resolver`; `Resolve(reqs []*chart.Dependency, repoNames map[string]string) (*chart.Lock, error)`; `HashReq(req, lock)`, `HashV2Req(req)` -> string; `GetLocalPath(repo, chartpath)` -> (string,error).
