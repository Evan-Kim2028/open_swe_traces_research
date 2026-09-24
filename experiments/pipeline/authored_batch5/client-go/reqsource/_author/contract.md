# Contract (L2) — reqsource

`GetRequestSource` returns the unknown label when the receiver is nil or
both type fields are empty, even when the internal flag is set. Otherwise
it joins the scope word, the type (with the unknown fallback for an empty
type), and the explicit type with underscores; whether a repeated explicit
type is deduplicated is not pinned, only that scope and type appear. The
internal check recognizes sources carrying the internal prefix and rejects
external, empty, and unknown sources. `IsRequestSourceInternal` is nil-safe
and follows the composed label; `BuildRequestSource` composes the label
from its arguments. The context helpers store a `RequestSource` value
under the exported key and the reader returns the unknown label when
absent. The resource-group helpers use a distinct context key and return
the empty string when unset.

| test | commitment |
| --- | --- |
| `TestDetail01` | Nil receiver and empty type fields return the unknown label, including when only the internal flag is set. |
| `TestDetail02` | The label is `{scope}_{type}` with an unknown-type fallback and a `_{explicit}` suffix when the explicit type differs; when explicit equals type only the scope-plus-type prefix is asserted. |
| `TestDetail03` | Sources carrying the internal prefix are internal; external, empty, and unknown sources are not — the prefix-vs-exact boundary is not pinned. |
| `TestDetail04` | `IsRequestSourceInternal` is false on nil and follows the composed label; `BuildRequestSource` produces the same label as the equivalent struct. |
| `TestDetail05` | The context helpers store a `RequestSource` value (not a pointer) under the exported key, and the reader returns the unknown label when absent. |
| `TestDetail06` | The resource-group helpers round-trip through their own context key and do not plant a request source. |
