# Exported API — auditstream

Constructor helpers building `AuditLogStreamConfig` values for each audit-log
streaming vendor supported by GitHub Enterprise.

- `NewAzureBlobStreamConfig(enabled, *AzureBlobConfig)`
- `NewAzureHubStreamConfig(enabled, *AzureHubConfig)`
- `NewAmazonS3OIDCStreamConfig(enabled, *AmazonS3OIDCConfig)`
- `NewAmazonS3AccessKeysStreamConfig(enabled, *AmazonS3AccessKeysConfig)`
- `NewSplunkStreamConfig(enabled, *SplunkConfig)`
- `NewHecStreamConfig(enabled, *HecConfig)`
- `NewGoogleCloudStreamConfig(enabled, *GoogleCloudConfig)`
- `NewDatadogStreamConfig(enabled, *DatadogConfig)`

Each returns `*AuditLogStreamConfig` carrying `Enabled`, a vendor `StreamType`
tag, and the vendor config under `VendorSpecific`.
