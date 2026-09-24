# Closure — auditstream

Package: github (root). File: github/enterprise_audit_log_stream.go.
Removed bodies: the 8 New*StreamConfig constructors — stubbed to
`&AuditLogStreamConfig{Enabled: enabled}` (drops StreamType tag and
VendorSpecific payload).
Kept: AuditLogStreamConfig struct, all vendor config structs and the
isAuditLogStreamVendorConfig marker set, GetAuditLogStreamKey.
Tests removed: 8 funcs in github/enterprise_audit_log_stream_test.go
(TestNew{AzureBlob,AzureHub,AmazonS3OIDC,AmazonS3AccessKeys,Splunk,Hec,
GoogleCloud,Datadog}StreamConfig).
