# Commitments — compat

1. Every `genpkg`-taking entry point calls `requireGeneratedPackage` before delegating to the planned file accessor. In-tree coverage: `TestReleasedHTTPFileFunctionsUsePlannedPackage` (trimmed). Inferable: doc — each GoDoc states "genpkg must match the package used to create data".
2. A mismatched package panics rather than silently returning files planned for a different package. In-tree coverage: `TestReleasedHTTPFileFunctionsUsePlannedPackage` (trimmed). Inferable: doc — "rejects a package argument that does not describe the HTTP data"; panic text shape only.
3. `PathFiles` takes no `genpkg` and performs no check. In-tree coverage: `TestReleasedHTTPFileFunctionsUsePlannedPackage` (trimmed). Inferable: doc — its signature is the released API.
