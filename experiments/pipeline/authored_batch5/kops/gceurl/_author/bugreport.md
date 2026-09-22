# Bug report

GCE URL and label helpers are broken: `BuildURL` omits `projects/` or the location segment,
`ParseGoogleCloudURL` accepts malformed URLs (wrong host, missing version, trailing junk) and
mis-parses the `TYPE/NAME` tail, `EncodeGCELabel` doesn't escape disallowed bytes (or escapes
alphanumerics), `DecodeGCELabel` can't reverse the encoding, and `TagForRole` produces tags
without the cluster prefix or role.

Expected: strict `compute/v1|beta` URL round-trip; `-XY` hex escaping of non-`[0-9a-z]` bytes;
decode reverses it; role tags of the form `<cluster>-k8s-io-role-<role>` truncated to 63.

Reproduce with:

```
go test -count=1 ./upup/pkg/fi/cloudup/gce/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
