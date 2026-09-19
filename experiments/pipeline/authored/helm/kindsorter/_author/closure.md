# Closure — kindsorter

Package: pkg/release/v1/util. Files: kind_sorter.go, manifest_sorter.go, sorter.go.

Removed: 12 functions (bodies stubbed to `panic("excised: <name>")`, signatures and doc comments preserved, compiles clean).

`SortManifests(files map[string]string, _ VersionSet, ordering KindSortOrder) (hooks []*release.Hook, manifests []Manifest, err error)`; `InstallOrder`/`UninstallOrder` KindSortOrder vars; `SortByName`/`SortByDate`/`SortByRevision`/`Reverse` on []*rspb.Release.
