# Contract (L2) — osmetadata

Local instance metadata is JSON (`name`, `meta.KubernetesCluster`, `project_id`, `availability_zone`, `hostname`, `uuid`).

Search order is a comma-separated list of source ids (`configDrive`, `metadataService`). Each id is trimmed. Sources are tried in order; the first success wins. If every source fails, the last error is returned. An unknown id is itself an error (and may be that last error).

Config drive: prefer `/dev/disk/by-label/config-2`; if missing, `blkid -l -t LABEL=config-2 -o device`. Mount iso9660 ro, then vfat ro on failure. Read `openstack/latest/meta_data.json` from the mount and decode JSON; unmount after.

Metadata service: HTTP GET the configured URL. Only status 200 is success; other statuses are errors. Body is the same JSON.

Default order used by the exported entry is config drive then metadata service.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestGetMetadataFromMetadataServiceReturnsNotFoundError` | non-200 HTTP is an error |
| `TestGetMetadataFromMetadataService` | 200 body decodes |
| `TestGetMetadataFromConfigDriveReturnsErrorWhenNoDeviceIsFound` | missing config-2 device/blkid fails |
| `TestGetMetadataFromConfigDrive` | successful mount+read |
| `TestGetMetadataReturnsLastErrorWhenNoMetadataWasFound` | last error after all sources fail |
| `TestGetMetadataFromConfigDriveWhenItIsFirstInSearchOrder` | first source wins |
| `TestGetMetadataFromServiceEndpointWhenConfigDriveFails` | fallback to HTTP |
| `TestGetMetadataFromServiceEndpointWhenItIsFirstInSearchOrder` | HTTP-first order |
| `TestGetMetadataFromConfigDriveWhenServiceEndpointFails` | fallback to config drive |
