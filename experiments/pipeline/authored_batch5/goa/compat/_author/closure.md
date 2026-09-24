# Closure — compat

`http/codegen/compatibility.go` — the released plugin-facing file
entry points (`ClientFiles`, `ServerFiles`, `ServerTypeFiles`,
`ClientTypeFiles`, `PathFiles`, `ClientEncodeDecodeFile`,
`ServerEncodeDecodeFile`, `WebsocketClientFile`) which delegate to the planned
transport file lists after checking the caller's `genpkg` argument matches the
package the `ServicesData` was created with.

Symbols stubbed: `ClientFiles`, `ServerFiles`, `ServerTypeFiles`, `ClientTypeFiles`, `PathFiles`, `ClientEncodeDecodeFile`, `ServerEncodeDecodeFile`, `WebsocketClientFile`, `requireGeneratedPackage`.

Tests removed (reach the stubs, verified by excision):
- `http/codegen/plugin_api_compatibility_test.go`: `TestReleasedHTTPFileFunctionsUsePlannedPackage`
