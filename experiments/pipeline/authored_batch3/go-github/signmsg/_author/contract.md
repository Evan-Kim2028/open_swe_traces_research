# Contract (L2) — signmsg

- The signing boundary is guarded: a request made with no signer carries no
  signature — no signature materializes from an absent signer. (The literal
  commitment — a nil signer rejected with an error — is unreachable through
  the public surface: a nil-interface signer bypasses signing per the visible
  option dispatch, and a typed-nil signer arrives as a non-nil interface.
  The suite asserts the reachable complement.)
- The payload handed to the signer is the Git object text: a `tree <sha>`
  line, one `parent <sha>` line per parent in order, an
  `author <name> <<email>> <unix> <±HHMM>` line, the same shape for
  `committer`, then a blank line, then the commit message — newline-joined.
  The field order and labels are the documented object convention; exact
  spacing beyond that convention is implementation detail.
- A nil committer falls back to the author — the committer line carries the
  author's identity.
- A commit missing required content (no/empty message, no author) is
  rejected before any serialization — the signer is never invoked and no
  request leaves the client.
- The signature attached to the request is exactly the byte string the
  signer produced for the payload — streamed through the signer's writer.

## Coverage

| hidden test | DETAILS row |
|---|---|
| `TestDetail01` | 1 — no signer → unsigned request (reachable complement; literal error-rejection unreachable black-box) |
| `TestDetail02` | 2 — payload shape: tree/parent/author/committer lines + blank line + message |
| `TestDetail03` | 3 — nil committer → author identity on the committer line |
| `TestDetail04` | 4 — missing fields rejected before serialization; signer never invoked (shape) |
| `TestDetail05` | 5 — request carries the signer's output verbatim |
