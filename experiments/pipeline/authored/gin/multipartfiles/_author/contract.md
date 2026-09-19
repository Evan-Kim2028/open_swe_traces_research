# Contract (L2) — multipartfiles

When a multipart request is mapped, each field is first checked against the uploaded file list for its form key: only when that key has files does file handling apply, otherwise ordinary form values are used. A *multipart.FileHeader field receives the first uploaded file; a multipart.FileHeader value field receives a copy of the first file header. A slice of file-header types is rebuilt to exactly the number of uploaded files; a fixed array must equal the file count or binding fails with the exported length error. Each element is assigned through the same per-element rules. A field kind that cannot hold a file (e.g. a plain int under a file key) returns the exported invalid-field-type error rather than silently skipping.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestFormMultipartBindingBindOneFile` | single file lands on *multipart.FileHeader fields and file-name fields |
| `TestFormMultipartBindingBindTwoFiles` | multiple files fill slice/array fields element-wise |
| `TestFormMultipartBindingBindError` | wrong field kind under a file key returns the exported error |
