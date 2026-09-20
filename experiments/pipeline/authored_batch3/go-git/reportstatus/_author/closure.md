# Closure — reportstatus

Package: `plumbing/protocol/packp`. File: `report_status.go`.

Removed (8 functions stubbed): `ReportStatus.Error`, `ReportStatus.Encode`, `ReportStatus.Decode`,
`ReportStatus.scanFirstLine`, `ReportStatus.decodeReportStatus`, `ReportStatus.decodeCommandStatus`,
`CommandStatus.Error`, `CommandStatus.encode`.

Kept: types `ReportStatus{UnpackStatus string; CommandStatuses []*CommandStatus}`,
`CommandStatus{ReferenceName, Status}`, error types `UnpackStatusErr`, `CommandStatusErr` (with
their `Error()` strings), the `ok` constant, shared wire constants and `ErrNilWriter`.

All 23 `*_test.go` files in the package deleted (suite covers this codec in
`report_status_test.go` plus `conformance_test.go`, `receive-pack` paths).
