# Closure — storage

Package: pkg/storage. Files: storage.go.

Removed: 12 functions (bodies stubbed to `panic("excised: <name>")`, signatures and doc comments preserved, compiles clean).

`Init(d driver.Driver) *Storage`. Methods: `Create(rls)`, `Update(rls)`, `Get(name, version)`, `Delete(name, version)`, `ListReleases`, `ListUninstalled`, `ListDeployed`, `Deployed(name)`, `DeployedAll(name)`, `History(name)`, `Last(name)`. `MaxHistory` const limits revisions per name.
