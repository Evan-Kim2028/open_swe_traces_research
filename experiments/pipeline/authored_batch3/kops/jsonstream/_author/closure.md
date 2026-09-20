# Closure — jsonstream

Package: `pkg/jsonutils` (`example.internal/clustkit/pkg/jsonutils`).

Files: `pkg/jsonutils/streamwriter.go` (184 lines, 4 funcs).

Removed functions (bodies stubbed): `JSONStreamWriter.Path`, `JSONStreamWriter.WriteToken`, `JSONStreamWriter.writeRaw`.

Exported entry point(s): `NewJSONStreamWriter(out)`, `(*JSONStreamWriter).WriteToken`, `(*JSONStreamWriter).Path` — re-emit a `json.Decoder` token stream as indented JSON while tracking the current field path.
