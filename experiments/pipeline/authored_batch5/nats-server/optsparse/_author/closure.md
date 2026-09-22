# Closure — optsparse

Package `server`, file `server/opts.go`.

Removed (stubbed): `parseDuration`, `parseWriteDeadlinePolicy`,
`trackExplicitVal`, `parseListen`, `parseURLs`, `parseURL`,
`getStorageSize`, `parseCompression`.

Retained: `unwrapValue`, `convertPanicToError`, `convertPanicToErrorList`
(the panic-recovery/unwrap machinery all parsers share),
`parseCluster`/`parseGateway`/`parseLeafNodes`/all other parse* drivers,
`hostPort`, `CompressionOpts`, `token`, `configErr`, `configWarningErr`,
`Options` struct and `ProcessConfigFile`.

No import changes.

Tests snipped in server/opts_test.go: `TestGetStorageSize`,
`TestListenConfig`, `TestListenPortOnlyConfig`,
`TestListenPortWithColonConfig`, `TestMalformedListenAddress`,
`TestMalformedClusterAddress`, `TestParseWriteDeadline`. No test files
deleted; hundreds of other config tests still drive the parsers through
ProcessConfigFile.
