# DETAILS — uploadreq

1. A `urlStr` containing `..` path segments is rejected before any
   request is built. Inferable: partially — rejecting traversal is
   derivable; the exact error identity is not (assert shape only).
2. An absolute `urlStr` naming a host that is not a configured origin is
   rejected; a configured upload origin is accepted. Inferable: doc —
   the file documents the destination rule on ErrUntrustedDestination.
3. The caller's reader is wrapped in a non-`io.Seeker` facade so the
   transport cannot observe concrete body types. Inferable: no — the
   type-hiding wrapper is an internal workaround; assert behavior only.
4. `GetBody` is set only when `reader` implements both `io.Seeker` and
   `io.ReaderAt`; it returns an independent rewindable view starting at
   the recorded offset. Inferable: partially — rewindable uploads are
   derivable from retry requirements; the seeker+readerAt conjunction
   and offset capture are arbitrary.
5. `GetBody` is absent for non-seekable readers (no buffering large
   uploads in memory). Inferable: partially.
6. `ContentLength` is set to `size`.
   Inferable: yes.
7. `mediaType` defaults when empty.
   Inferable: doc — the default media type is a documented constant.
