# API left after excision

```go
func dnsAltNameMatches(dnsAltNameLabels []string, urls []*url.URL) bool
```

`dnsAltNameLabels` is the already-split, already-lowercased SAN (see `dnsAltNameLabels`). `urls` are candidate connect URLs whose hostnames are tested against that SAN.
