# Closure — gceurl

Package: `upup/pkg/fi/cloudup/gce` (`example.internal/clustkit/upup/pkg/fi/cloudup/gce`).

Files: `gce_url.go` (2 funcs), `labels.go` (3 funcs) — 5 funcs.

Removed functions (bodies stubbed): `GoogleCloudURL.BuildURL`, `ParseGoogleCloudURL`,
`EncodeGCELabel`, `DecodeGCELabel`, `TagForRole`.

Exported entry point(s): `ParseGoogleCloudURL`/`BuildURL` build the `selfLink` URLs used
throughout the GCE provider; `EncodeGCELabel`/`TagForRole` produce resource labels/tags.

Test files removed in excision: none.
