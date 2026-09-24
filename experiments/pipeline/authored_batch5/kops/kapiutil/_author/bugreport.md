# Bug report

Node role detection and taint/version parsing are broken: `GetNodeRole` no longer recognises the
`node-role.kubernetes.io/*` labels, `ParseTaint` returns malformed maps (missing keys) or never
errors, `ParseKubernetesVersion` rejects tolerable inputs like `v1.28` or `.../v1.28.txt` URLs,
`IsKubernetesGTE` compares pre-release fields it should ignore, and `ParseVersion` accepts sloppy
versions it should reject.

Expected: roles resolve from the well-known label keys with master before control-plane before
node before api-server; taints parse `key[=value]:effect` into a 3-key map; version parsing is
tolerant (with the `/v1.N.` URL fallback) for kubernetes versions but strict for `ParseVersion`.

Reproduce with:

```
go test -count=1 ./pkg/apis/kops/util/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
