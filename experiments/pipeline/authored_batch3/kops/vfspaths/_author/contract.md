# Contract (L2) — vfspaths

`BuildVfsPath` dispatches on scheme: no scheme/`file://`→FSPath; `s3`/`do`/`linode`/`hos`/`scw`→S3Path (only `s3` works without `S3_ENDPOINT`; the rest require it, linode adds checksum pins); `memfs` needs an initialized context; `gs`/`k8s`/`swift` build their path types with host=bucket, path=key; `azureblob` requires `AZURE_STORAGE_ACCOUNT` to be UNSET and splits host=account, first segment=container, rest=key; unknown schemes error. Every cloud builder validates scheme and rejects empty buckets. `RetryWithBackoff` runs the condition before the first sleep, attempts up to `Steps`, and returns the last (done,err); `nextBackoffDuration` scales by Factor clamped to Cap>0.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestNextBackoffDuration` | factor growth with cap clamping |
| `TestRetryWithBackoffRespectsCap` | sleep bounded by cap; attempts counted |
| `Test_S3Path_Parse`, `Test_LinodePath_Parse`, `Test_NonLinodeObjectStoragePaths_HaveCorrectScheme` | s3/linode path construction + scheme validation |
| `TestAzureBlobPath*`, `TestBuildAzureBlobPath` | azureblob account/container/key split + env rule |
| `Test_GSPath_Parse` | gs path construction |
