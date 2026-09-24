# Contract (L2) — patopts

- Each `Owner` entry becomes its own `owner[]=<value>` query pair, order
  preserved. (The `[]` spelling is the documented wire convention kept in
  the surviving option comment.)
- Each `TokenID` entry becomes its own `token_id[]=<decimal>` query pair,
  order preserved.
- The array parameters join the existing query rather than replacing it —
  generic option parameters and the array parameters coexist in the final
  query string.
- A nil options value returns an error and no request is made.
- The remaining option fields flow through the generic query encoder under
  their declared parameter names.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — `owner[]` repeated per entry, order preserved |
| `TestDetail02` | 2 — `token_id[]` repeated per entry, decimal values |
| `TestDetail03` | 3 — array params coexist with generic params |
| `TestDetail04` | 4 — nil opts → error, no request (shape) |
| `TestDetail05` | 5 — generic encoder still emits other option fields |
