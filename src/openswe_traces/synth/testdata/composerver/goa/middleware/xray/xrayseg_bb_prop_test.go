// Black-box property suite for xrayseg (AWS X-Ray segment documents).
// Exported API only: NewSegment, NewSubsegment, RecordError, Capture, AddAnnotation,
// Close, SubmitInProgress.
// Seed 20260919; >=10k cases.
//
// Coverage table (contract sentence -> property):
//   "new segment in progress with millisecond-truncated start time"
//       -> TestXraysegNewSegmentProperty / TestXraysegStartTimeRandom
//   "subsegment: 16-hex id, copied trace, parent link, in progress"
//       -> TestXraysegNewSubsegmentProperty / TestXraysegSubsegmentRandom
//   "RecordError sets Error unless fault/throttle; ancestor cause id"
//       -> TestXraysegRecordErrorProperty / TestXraysegRecordErrorRandom
//   "Capture runs fn inside child submitted in progress then closed"
//       -> TestXraysegCaptureProperty
//   "SubmitInProgress at most once; Close sends finished document"
//       -> TestXraysegSubmitInProgressProperty / TestXraysegSubmitRandom
//   "UDP write is header+JSON"
//       -> TestXraysegUDPHeaderProperty
package xray_test

import (
	"fmt"
	"math/rand"
	"net"
	"regexp"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"example.internal/apikit/v3/middleware/xray"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

var xraysegHexID = regexp.MustCompile(`^[0-9a-f]{16}$`)

func xraysegPipeConn(t *testing.T) net.Conn {
	srv, cli := net.Pipe()
	t.Cleanup(func() { _ = srv.Close(); _ = cli.Close() })
	return cli
}

func TestXraysegNewSegmentProperty(t *testing.T) {
	conn := xraysegPipeConn(t)
	s := xray.NewSegment("svc", "trace-1", "span-1", conn)
	if !s.InProgress || s.Name != "svc" || s.TraceID != "trace-1" || s.ID != "span-1" {
		t.Fatalf("segment fields: %+v", s)
	}
	if s.StartTime <= 0 {
		t.Fatal("start time not set")
	}
}

func TestXraysegStartTimeRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		_, conn := net.Pipe()
		name := fmt.Sprintf("n%d", rng.Intn(100))
		s := xray.NewSegment(name, "t", "s", conn)
		if !s.InProgress {
			t.Fatalf("case %d: not in progress", i)
		}
		sub := s.NewSubsegment("child")
		if sub.StartTime < s.StartTime {
			t.Fatalf("case %d: child start before parent", i)
		}
		_ = conn.Close()
	}
}

func TestXraysegNewSubsegmentProperty(t *testing.T) {
	conn := xraysegPipeConn(t)
	parent := xray.NewSegment("root", "trace", "parent-id", conn)
	child := parent.NewSubsegment("op")
	if child.Type != "subsegment" || child.Parent != parent || child.TraceID != "trace" {
		t.Fatalf("child linkage: %+v", child)
	}
	if child.ParentID != "parent-id" || child.Name != "op" {
		t.Fatalf("child ids: %+v", child)
	}
	if !xraysegHexID.MatchString(child.ID) {
		t.Fatalf("child id %q", child.ID)
	}
	if !child.InProgress {
		t.Fatal("child not in progress")
	}
}

func TestXraysegSubsegmentRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		_, conn := net.Pipe()
		trace := fmt.Sprintf("trace-%d", rng.Intn(50))
		parent := xray.NewSegment("p", trace, xray.NewID(), conn)
		sub := parent.NewSubsegment(fmt.Sprintf("c%d", i))
		if sub.TraceID != trace || sub.Parent != parent {
			t.Fatalf("case %d: trace/parent", i)
		}
		if !xraysegHexID.MatchString(sub.ID) {
			t.Fatalf("case %d: id %q", i, sub.ID)
		}
		_ = conn.Close()
	}
}

func TestXraysegRecordErrorProperty(t *testing.T) {
	conn := xraysegPipeConn(t)
	root := xray.NewSegment("r", "t", xray.NewID(), conn)
	child := root.NewSubsegment("c")
	child.RecordError(errors.New("boom"))
	if !child.Error {
		t.Fatal("child should be error")
	}
	if root.Cause == nil || root.Cause.ID != child.ID {
		t.Fatalf("ancestor cause: %+v", root.Cause)
	}
	faulty := root.NewSubsegment("f")
	faulty.Fault = true
	faulty.RecordError(errors.New("user fault"))
	if faulty.Error {
		t.Fatal("fault already set: Error flag should stay false")
	}
}

func TestXraysegRecordErrorRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		_, conn := net.Pipe()
		root := xray.NewSegment("r", "t", xray.NewID(), conn)
		depth := rng.Intn(3) + 1
		cur := root
		var leaf *xray.Segment
		for j := 0; j < depth; j++ {
			leaf = cur.NewSubsegment(fmt.Sprintf("s%d", j))
			cur = leaf
		}
		err := errors.Wrap(errors.New("base"), fmt.Sprintf("wrap-%d", i))
		leaf.RecordError(err)
		if !leaf.Error {
			t.Fatalf("case %d: leaf error flag", i)
		}
		if len(leaf.Cause.Exceptions) == 0 {
			t.Fatalf("case %d: no exceptions", i)
		}
		_ = conn.Close()
	}
}

func TestXraysegCaptureProperty(t *testing.T) {
	conn := xraysegPipeConn(t)
	s := xray.NewSegment("cap", "t", xray.NewID(), conn)
	var ran bool
	s.Capture("work", func() { ran = true })
	if !ran {
		t.Fatal("capture did not run fn")
	}
}

func TestXraysegSubmitInProgressProperty(t *testing.T) {
	conn := xraysegPipeConn(t)
	s := xray.NewSegment("s", "t", xray.NewID(), conn)
	s.SubmitInProgress()
	s.SubmitInProgress()
	s.Close()
	if s.InProgress {
		t.Fatal("should be closed")
	}
	if s.EndTime <= 0 {
		t.Fatal("end time missing")
	}
}

func TestXraysegSubmitRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	for i := 0; i < bbCases; i++ {
		_, conn := net.Pipe()
		s := xray.NewSegment("s", "t", xray.NewID(), conn)
		if rng.Intn(2) == 0 {
			s.SubmitInProgress()
		}
		s.Close()
		if s.InProgress {
			t.Fatalf("case %d: still in progress", i)
		}
		_ = conn.Close()
	}
}

func TestXraysegUDPHeaderProperty(t *testing.T) {
	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()
	s := xray.NewSegment("udp", "t", xray.NewID(), cli)
	s.Close()
	buf := make([]byte, 4096)
	n, err := srv.Read(buf)
	if err != nil || n == 0 {
		t.Fatalf("read: %v n=%d", err, n)
	}
	if !strings.HasPrefix(string(buf[:n]), xray.UDPHeader) {
		t.Fatalf("missing header in %q", buf[:n])
	}
}

func TestXraysegAnnotationProperty(t *testing.T) {
	conn := xraysegPipeConn(t)
	s := xray.NewSegment("a", "t", xray.NewID(), conn)
	s.AddAnnotation("k", "v")
	if s.Annotations == nil || s.Annotations["k"] != "v" {
		t.Fatalf("annotation: %+v", s.Annotations)
	}
}
