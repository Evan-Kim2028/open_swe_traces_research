# Exported API — kindsorter

`SortManifests(files map[string]string, _ VersionSet, ordering KindSortOrder) (hooks []*release.Hook, manifests []Manifest, err error)`; `InstallOrder`/`UninstallOrder` KindSortOrder vars; `SortByName`/`SortByDate`/`SortByRevision`/`Reverse` on []*rspb.Release.
Callers: pkg/action install/upgrade/uninstall manifest ordering.
