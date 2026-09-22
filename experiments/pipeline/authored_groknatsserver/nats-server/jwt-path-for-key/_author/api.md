# API left after excision

```go
func (store *DirJWTStore) pathForKey(publicKey string) string
```

Returns a filesystem path for the JWT identified by `publicKey`, or empty when the key is rejected. `store.directory`, `store.shard`, and `fileExtension` remain.
