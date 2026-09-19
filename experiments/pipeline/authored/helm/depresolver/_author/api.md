# Exported API — depresolver

`New(chartpath, cachepath, registryClient) *Resolver`; `Resolve(reqs []*chart.Dependency, repoNames map[string]string) (*chart.Lock, error)`; `HashReq(req, lock)`, `HashV2Req(req)` -> string; `GetLocalPath(repo, chartpath)` -> (string,error).
Callers: pkg/downloader/manager dependency update/build.
