# Closure — ratecat

Package: github (root). File: github/github.go.
Removed body: GetRateLimitCategory — stubbed to always return CoreCategory.
Kept: RateLimitCategory constants, checkRateLimitBeforeDo /
checkSecondaryRateLimitBeforeDo machinery, Response.Rate plumbing.
Tests removed: TestDo_rateLimitCategory in github/github_test.go;
TestCodeScanningService_UploadSarif in github/code_scanning_test.go;
TestEnterpriseService_GetAuditLog in github/enterprise_audit_log_test.go;
TestOrganizationService_GetAuditLog in github/orgs_audit_log_test.go —
each asserts the response rate-limit headers land in the non-core bucket.
