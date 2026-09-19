# Exported API — xrayseg

```
func NewSegment(name, traceID, spanID string, conn net.Conn) *Segment
func (s *Segment) NewSubsegment(name string) *Segment
func (s *Segment) RecordError(e error)
func (s *Segment) Capture(name string, fn func())
func (s *Segment) AddAnnotation(key, value string)
func (s *Segment) Close()
func (s *Segment) SubmitInProgress()
```

NewSegment starts in-progress with millisecond-truncated start time. NewSubsegment copies trace id, sets type subsegment, parent, and a fresh id. RecordError appends an exception (cause + stack when the error was built with pkg/errors) and marks Error unless Fault/Throttle already set; ancestors get a cause id. Capture opens a subsegment, submits in-progress, runs fn, closes. Close sets end time, clears in-progress, writes UDP. SubmitInProgress flushes at most once while still in progress. Each UDP write is header+JSON in one Write.

## Pre-existing callers

http/middleware/xray, grpc/middleware/xray, Connect tests, segment tests.
