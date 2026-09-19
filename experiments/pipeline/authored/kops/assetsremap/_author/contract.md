# Contract (L2) — assetsremap

Image rewrite has two optional layers from the assets spec, applied in order.

Container proxy: if set (trailing slash stripped), a name with at most one `/` whose first segment has no `.` or `:` is treated as a hub image and the proxy is prepended. Otherwise the first path segment (registry host) is replaced with the proxy. Tags/digests on the remainder are kept.

Container registry: if set, strip a leading `registry.k8s.io/`, then if the name does not already start with `registry/`, replace every `/` with `-` and prefix `registry/`. A second pass must not double-prefix (spec assembly calls this until the cluster spec converges).

Three well-known image prefixes may be overridden by env (`DNSCONTROLLER_IMAGE`, `KOPSCONTROLLER_IMAGE`, `KUBE_APISERVER_HEALTHCHECK_IMAGE`) before normalize. After normalize the image is appended to the in-memory image list. Digest suffix `@sha` is added only when a digest resolver is installed, the ImageDigest feature is on, `KOPS_BASE_URL` is empty, and the name does not already contain `@`; resolver errors leave the undigested name.

File remap: a nil URL is an error. If a file repository is set, join repository path with the canonical path; commas in the escaped path become `%2C`. Hash: use the caller’s hash if given, else well-known asset hashes, else a process-wide cache keyed by the URL actually fetched (canonical URL when `getAssets` is true, otherwise the download URL). Cache stores only successful hashes, never failures. Hash files tried are `.sha256` then `.sha256sum` across URL mirrors, with short backoff; the first field of the file is the digest. Concurrent RemapImage/RemapFile must not race; ImageAssets/FileAssets return a copy sorted by canonical location/URL.

Manifest remap loads YAML objects, remaps each image, writes YAML back.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestValidate_RemapImage_ContainerProxy_AppliesToDockerHub` | hub image (no registry host) gets proxy prepended |
| `TestValidate_RemapImage_ContainerProxy_AppliesToSimplifiedDockerHub` | org/name hub form also prepends |
| `TestValidate_RemapImage_ContainerProxy_AppliesToSimplifiedKubernetesURL` | dotted first segment is a host and is replaced |
| `TestValidate_RemapImage_ContainerProxy_AppliesToLegacyKubernetesURL` | legacy k8s registry host is replaced |
| `TestValidate_RemapImage_ContainerProxy_AppliesToImagesWithTags` | tags survive proxy rewrite |
| `TestValidate_RemapImage_ContainerRegistry_MappingMultipleTimesConverges` | second registry pass does not double-prefix |
| `TestRemapURLPathDelimiterEscaping` | commas in file paths are `%2C` |
| `TestRemapEmptySection` | empty YAML section does not panic |
| `TestAssetBuilderConcurrentCollection` | concurrent remap + sorted snapshot getters |
| `TestFindHashCachesDownloadedHashesByResolvedURL` | successful hash downloads are cached by resolved URL |
| `TestFindHashDoesNotCacheFailures` | failed hash lookups are not cached |
