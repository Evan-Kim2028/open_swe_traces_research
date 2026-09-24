# Closure — fetchsbom

Package: github (root). File: github/dependency_graph.go.
Removed bodies: FetchSBOM and fetchSBOMFromURL — stubbed to variants
that stop at the redirect: FetchSBOM always returns `loc.String()`
ignoring `followRedirectsClient`, and the helper decodes the body
without `CheckResponse` validation and without capturing/closing the
original network body. The "expected redirect" guard and the SBOM JSON
decode keep working.
Kept: GenerateSBOM, SBOM/SBOMInfo types, Response.
Tests removed: 2 funcs in github/dependency_graph_test.go
(TestDependencyGraphService_FetchSBOM_Download,
_DownloadTransportError).
