# Exported API — jsonstream

Package `pkg/jsonutils` (importable as `example.internal/clustkit/pkg/jsonutils`).

- `func NewJSONStreamWriter(out io.Writer) *JSONStreamWriter` — construct a writer around `out`.
- `func (j *JSONStreamWriter) WriteToken(token json.Token) error` — write the next token (delimiters, bool, string, float64, json.Number, nil).
- `func (j *JSONStreamWriter) Path() string` — the dot-joined path of enclosing field names at the current position.

Production callers: `tests/fuzz/fuzz.go`; siblings in `pkg/jsonutils` (`transform.go`) remain intact.
