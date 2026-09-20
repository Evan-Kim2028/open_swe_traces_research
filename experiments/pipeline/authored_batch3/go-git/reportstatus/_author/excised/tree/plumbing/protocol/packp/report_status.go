package packp

import (
	_ "bytes"
	"fmt"
	"io"
	_ "strings"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/pktline"
)

const (
	ok = "ok"
)

// UnpackStatusErr is the error returned when the report status is not ok.
type UnpackStatusErr struct {
	Status string
}

// Error implements the error interface.
func (e UnpackStatusErr) Error() string {
	return fmt.Sprintf("unpack error: %s", e.Status)
}

// CommandStatusErr is the error returned when the command status is not ok.
type CommandStatusErr struct {
	ReferenceName plumbing.ReferenceName
	Status        string
}

// Error implements the error interface.
func (e CommandStatusErr) Error() string {
	return fmt.Sprintf("command error on %s: %s", e.ReferenceName.String(), e.Status)
}

// ReportStatus is a report status message, as used in the git-receive-pack
// process whenever the 'report-status' capability is negotiated.
// The zero value is safe to use.
type ReportStatus struct {
	UnpackStatus    string
	CommandStatuses []*CommandStatus
}

// Error returns the first error if any.
func (s *ReportStatus) Error() error {
	panic("excised: ReportStatus.Error")
}

// Encode writes the report status to a writer.
func (s *ReportStatus) Encode(w io.Writer) error {
	panic("excised: ReportStatus.Encode")
}

// Decode reads from the given reader and decodes a report-status message. It
// does not read more input than what is needed to fill the report status.
func (s *ReportStatus) Decode(r io.Reader) error {
	panic("excised: ReportStatus.Decode")
}

func (s *ReportStatus) scanFirstLine(sc *pktline.Scanner) ([]byte, error) {
	panic("excised: ReportStatus.scanFirstLine")
}

func (s *ReportStatus) decodeReportStatus(b []byte) error {
	panic("excised: ReportStatus.decodeReportStatus")
}

func (s *ReportStatus) decodeCommandStatus(b []byte) error {
	panic("excised: ReportStatus.decodeCommandStatus")
}

// CommandStatus is the status of a reference in a report status.
// See ReportStatus struct.
type CommandStatus struct {
	ReferenceName plumbing.ReferenceName
	Status        string
}

// Error returns the error, if any.
func (s *CommandStatus) Error() error {
	panic("excised: CommandStatus.Error")
}

func (s *CommandStatus) encode(w io.Writer) error {
	panic("excised: CommandStatus.encode")
}
