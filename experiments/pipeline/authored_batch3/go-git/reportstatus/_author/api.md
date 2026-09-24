# Exported API — reportstatus

Package `plumbing/protocol/packp` (module `example.internal/gitkit/v6`).

Types kept: `ReportStatus{UnpackStatus string; CommandStatuses []*CommandStatus}`,
`CommandStatus{ReferenceName plumbing.ReferenceName; Status string}`,
`UnpackStatusErr{Status string}`, `CommandStatusErr{ReferenceName, Status}` (both implement
`error` with their kept `Error()` methods).

Methods: `(*ReportStatus).Decode(r io.Reader) error`, `(*ReportStatus).Encode(w io.Writer) error`,
`(*ReportStatus).Error() error`, `(*CommandStatus).Error() error`.

Unexported helpers (stubbed): `ReportStatus.scanFirstLine`, `ReportStatus.decodeReportStatus`,
`ReportStatus.decodeCommandStatus`, `CommandStatus.encode`.

Callers: `receive-pack`/`send-pack` push flows — the report-status message is read after the pack
when the `report-status` capability is negotiated. In-tree tests removed: whole `packp` suite.
