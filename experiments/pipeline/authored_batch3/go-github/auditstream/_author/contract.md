# Contract (L2) — auditstream

- Every audit-stream constructor copies its `enabled` argument into the
  returned configuration's `Enabled` field.
- Every constructor stores its vendor-config argument in `VendorSpecific`
  unchanged — the very value passed, same type and identity.
- Each constructor emits a non-empty, vendor-distinct `StreamType` tag; the
  literal spellings are implementation detail and are not part of the
  contract. What is contracted: non-empty, and distinct across vendor
  families.
- The two Amazon S3 constructors (OIDC and access-keys) emit the *same*
  `StreamType`; they differ only in the concrete type stored under
  `VendorSpecific`.
- No constructor validates or mutates the vendor config: a nil `cfg` argument
  is accepted without panic and stored as a typed nil of that vendor's
  config type.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — `Enabled` mirrors the argument for all eight constructors, both true and false |
| `TestDetail02` | 2 — `VendorSpecific` is the passed pointer, per constructor |
| `TestDetail03` | 3 — non-empty `StreamType`, pairwise-distinct across vendor families (shape; literals not pinned) |
| `TestDetail04` | 4 — S3 pair shares `StreamType`, differs only in `VendorSpecific` concrete type (shape) |
| `TestDetail05` | 5 — nil vendor config never panics and is stored as a typed nil |
