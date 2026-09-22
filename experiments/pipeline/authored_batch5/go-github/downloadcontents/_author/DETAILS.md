# DETAILS — downloadcontents

1. When the API response carries a `download_url`, its bytes are fetched
   with a plain GET through the client's own `http.Client` and the body
   is returned to the caller. Inferable: doc — the method exists to
   resolve download_url; that credentials stay origin-scoped is the
   documented client behavior.
2. When `download_url` is absent and the file has no inline `Content`,
   the call returns `ErrContentsNoDownloadURL` with the metadata still
   populated. Inferable: partially — the sentinel exists; which error
   wins when both are absent is arbitrary.
3. A directory or submodule path returns early without any download.
   Inferable: yes — directories have no downloadable bytes.
4. A file whose `Content` is populated inline returns that content
   wrapped as a reader without a second request. Inferable: partially.
5. On download transport error, the returned `*Response` wraps the
   download response (not the metadata response). Inferable: no — which
   response object propagates is an arbitrary choice; assert a response
   is returned, not which.
