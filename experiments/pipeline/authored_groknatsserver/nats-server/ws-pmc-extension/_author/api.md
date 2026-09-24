# API left after excision

```go
func wsPMCExtensionSupport(header http.Header, checkPMCOnly bool) (bool, bool)
```

Returns `(permessage-deflate offered, both no-context-takeover parameters present)`.

Constants still in the file: `wsPMCExtension`, `wsPMCSrvNoCtx`, `wsPMCCliNoCtx`.
