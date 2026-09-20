# Details — fidownload

1. `DownloadURL` with a non-nil hash first checks the EXISTING destination — a match returns the expected hash without any network traffic. Inferable: partially — doc implies caching, mechanism is a choice.
2. `downloadURLToWriter` dispatches on scheme: `gs`/`s3`/`azureblob` go through `vfs.Context.BuildVfsPath` + `WriteToWithContext` (a non-implementing path type is an error); every other scheme goes through `OpenURL` + copy. Inferable: no — the cloud-scheme set is arbitrary.
3. All downloaded bytes are hashed as they stream — sha256 by default or the expected hash's algorithm — and a mismatch is an error AFTER the bytes are written. Inferable: partially.
4. `downloadURLToFile` stages into a temp file `.<base>.tmp` in the destination dir, chmods 0644, then renames; the temp is removed on failure. Inferable: partially.
5. `OpenURL` uses a hardened client (dial/TLS/response-header/idle timeouts, proxy-from-environment) with a 3-minute request context; non-2xx (after redirects) is an error carrying the status; the returned reader cancels the context on Close. Inferable: no — timeout constants are arbitrary.
6. `cancelOnCloseReadCloser.Close` closes the body first, then cancels — Close still returns the body's error. Inferable: yes.
