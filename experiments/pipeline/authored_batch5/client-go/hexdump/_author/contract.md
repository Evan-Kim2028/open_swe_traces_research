# Contract (L2) — hexdump

`Hex` wraps a proto message so that formatting it yields a brace-enclosed
list of `Field:value` pairs separated by single spaces; internal `XXX_*`
fields are never rendered, and nested message fields recurse into the same
brace form. Byte-slice fields render as lowercase hex; the hex encoding
applies only when the slice element kind is `uint8` — slices of byte slices
render their inner bytes as text and other slice kinds render their
elements without hex encoding. A nil pointer field renders as `<nil>`, and
a nil message argument renders as `<nil>` too. Scalar and enum fields print
with the `%v` verb, so proto enums render by their declared names.

| test | commitment |
| --- | --- |
| `TestDetail01` | The output is brace-enclosed, contains `Field:value` pairs separated by spaces, contains no `XXX_*` internals, and nested messages recurse into braces — shape only, no exact layout literal. |
| `TestDetail02` | A byte slice renders as lowercase hex of its content; a slice of byte slices renders inner bytes as text rather than hex; a non-byte slice renders its elements without hex — shape only, the fallback rendering is not pinned. |
| `TestDetail03` | A nil message argument renders as `<nil>` and a nil pointer field renders as `<nil>`. |
| `TestDetail04` | Enum fields render by their declared names. |
