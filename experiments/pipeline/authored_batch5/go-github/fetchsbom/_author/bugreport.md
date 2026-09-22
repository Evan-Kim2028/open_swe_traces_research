# Bug report — fetchsbom

`DependencyGraphService.FetchSBOM` never returns the SBOM: when GitHub
redirects to the pre-signed download URL, the method hands back the
redirect string even when a `followRedirectsClient` was supplied, so
callers get no SBOM data. Redirect responses that fail are also not
validated — an error page would be decoded as SBOM JSON.

Expected: with a follow client, the pre-signed URL is fetched through
that credential-free client, validated, decoded, and returned as
`*SBOM`.

Got: `redirectURL` is always returned; the download path is dead code.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
