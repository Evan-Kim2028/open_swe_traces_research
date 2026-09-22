# API left after excision

```go
func (req *UploadRequest) Encode(w io.Writer) error
```

`UploadRequest` still has `Capabilities`, `Wants`, `Shallows`, `Depth`, and `Filter`. `DepthRequest.IsZero` is intact. `ErrDeepenMutuallyExclusive` lives in the decoder file.
