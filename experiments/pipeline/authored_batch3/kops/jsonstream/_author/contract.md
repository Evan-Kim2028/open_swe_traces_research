# Contract (L2) — jsonstream

`JSONStreamWriter` emits a JSON document incrementally: `WriteToken` writes the next raw token at the correct position, inserting a comma separator between siblings of the same object/array and a `:` inside the `key: value` structure. `Path` reports the current location. The writer tracks container nesting so the first element in a container gets no comma while subsequent elements do; a token written at top level after a completed document is an error state (no implicit second document).

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestWriteToken` | comma/colon insertion across nested objects and arrays |
