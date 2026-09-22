# Closure — erris

Package: github (root). File: github/github.go.
Removed bodies: Is() of ErrorResponse, RateLimitError, AbuseRateLimitError,
RedirectionError, AcceptedError; helpers compareHTTPResponse and
equalDurationPtr — stubbed to constant true/false.
Kept: error types, Error() methods, CheckResponse machinery.
Tests removed: 13 funcs in github/github_test.go — TestErrorResponse_Is,
TestRateLimitError_Is, TestAbuseRateLimitError_Is, TestAcceptedError_Is,
TestCompareHttpResponse, TestCheckResponse, TestCheckResponse_RateLimit,
TestCheckResponse_AbuseRateLimit,
TestCheckResponse_RateLimit_TooManyRequests,
TestCheckResponse_AbuseRateLimit_TooManyRequests,
TestCheckResponse_RedirectionError, TestCheckResponse_noBody,
TestCheckResponse_unexpectedErrorStructure; plus
TestDependencyGraphService_FetchSBOM_Download in
github/dependency_graph_test.go (uses errors.Is on download errors).
