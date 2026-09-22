# Bug report — downloadasset

`DownloadReleaseAsset` never follows redirects: even when the caller
supplies a `followRedirectsClient`, the method returns the redirect
URL instead of fetching it, so release-asset downloads on redirecting
endpoints return no data.

Expected: with a follow client, the redirect target is fetched through
that client and its validated body is streamed back; without one, the
redirect URL is returned for the caller to handle.

Got: `redirectURL` is always returned on redirect; a supplied follow
client is ignored, and redirect-fetched error responses leak their
original body.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
