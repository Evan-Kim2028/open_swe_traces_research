# Contract — advrefs-encode

Hidden suite: `tests/hidden/plumbing/protocol/packp/advrefs_encode_bb_test.go`
(package `packp`, in-package). One `TestDetailNN` per DETAILS.md line.
Wire output is framed with the intact `pktline` package and compared as
decoded payloads plus frame lengths.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | Encoding with protocol v1 emits a leading `version 1` pkt-line before any ref; v0 emits no version line; an unsupported version returns an error. |
| TestDetail02 | 2 | no | SHAPE: with a non-peeled HEAD present, the first advertisement line is HEAD; with HEAD absent, the first line is the first non-peeled ref in input order (not sorted order). |
| TestDetail03 | 3 | no | SHAPE: an empty reference set still produces a first line whose name is `capabilities^{}` and whose hash is all zero. |
| TestDetail04 | 4 | no | SHAPE: the first line payload is `<hash> SP <name> NUL <caps>` with the NUL present even when the capability list is empty. |
| TestDetail05 | 5 | doc | The remaining non-peeled refs are emitted sorted by name as `<hash> SP <name>`; the ref already used as the first line is not repeated. |
| TestDetail06 | 6 | doc | A peeled ref `name^{}` is written immediately after its base `name` line, ahead of refs that would sort between them. |
| TestDetail07 | 7 | yes | Shallow lines come after all ref lines as `shallow <hash>`, sorted by hex string. |
| TestDetail08 | 8 | yes | The advertisement's last packet is a flush (zero-length) packet. |
| TestDetail09 | 9 | doc | Every non-flush payload ends with a newline. |

Refusals/softening: line 2 is asserted only on the observable ordering (HEAD
first when present; first-in-input-order otherwise) — hash bookkeeping around
the first line is left free. Line 6 deliberately uses a peeled ref whose base
is not the first line; the first-line/peeled interaction is an undocumented
edge and is not asserted.
