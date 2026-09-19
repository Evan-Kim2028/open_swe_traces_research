# Exported API — storage

`Init(d driver.Driver) *Storage`. Methods: `Create(rls)`, `Update(rls)`, `Get(name, version)`, `Delete(name, version)`, `ListReleases`, `ListUninstalled`, `ListDeployed`, `Deployed(name)`, `DeployedAll(name)`, `History(name)`, `Last(name)`. `MaxHistory` const limits revisions per name.
Callers: pkg/action install/upgrade/rollback/uninstall via cfg.Storage.
