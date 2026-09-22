# DETAILS — uploadasset

1. A directory passed as `file` is rejected before any request.
   Inferable: partially — refusing a directory is derivable; the error
   literal is arbitrary (assert an error, not text).
2. A nil `reader`, negative `size`, or a release with empty `UploadURL`
   is rejected. Inferable: partially — guards on obviously-invalid
   input are derivable.
3. `release.UploadURL`'s `{?name,label}` URI-template suffix is stripped
   before the upload URL is used. Inferable: doc — the API hands back a
   template; stripping is the only way to use it.
4. A relative upload URL is normalized (leading `/` trimmed) so it
   composes with `Client.BaseURL` path prefixes; absolute URLs pass
   through `NewUploadRequest`'s destination gate. Inferable: partially.
5. Media type resolution: `opts.MediaType` wins; otherwise the
   extension of the file name (or `opts.Name`) is mapped through
   `mime.TypeByExtension`. Inferable: no — the precedence chain is
   arbitrary; assert a media type is produced, not which.
