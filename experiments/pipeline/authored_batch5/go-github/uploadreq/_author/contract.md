# Contract — uploadreq

`Client.NewUploadRequest` builds POST upload requests gated to configured
destinations, with traversal rejection and a body facade. Every commitment
below is covered by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **Path traversal rejected (shape).** A `urlStr` containing `..` path
   segments — literal or percent-encoded — is rejected with an error before
   any request is built; which error is not pinned. Covered by
   `TestDetail01`.
2. **Destination gate.** An absolute `urlStr` naming a non-configured host
   is rejected with `ErrUntrustedDestination`; a configured upload origin is
   accepted. Covered by `TestDetail02`.
3. **Body facade (shape).** The request body does not expose the caller's
   concrete reader type — a seekable caller input is hidden behind a
   non-`io.Seeker` facade. Covered by `TestDetail03`.
4. **Rewindable bodies.** `GetBody` is set when the reader supports
   rewinding (`io.Seeker` + `io.ReaderAt`) and returns an independent view
   starting at the offset the reader had when the request was built.
   Covered by `TestDetail04`.
5. **Non-seekable readers.** `GetBody` is absent for a non-seekable reader.
   Covered by `TestDetail05`.
6. **ContentLength.** `ContentLength` is set to `size`. Covered by
   `TestDetail06`.
7. **Media type default.** An empty `mediaType` defaults to the documented
   default media type; a given media type is honored. Covered by
   `TestDetail07`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — shape only |
| TestDetail02 | 2 | doc |
| TestDetail03 | 3 | no — shape only |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | partially |
| TestDetail06 | 6 | yes |
| TestDetail07 | 7 | doc |
