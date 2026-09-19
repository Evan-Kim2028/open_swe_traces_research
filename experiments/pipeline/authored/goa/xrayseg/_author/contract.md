# Contract (L2) — xrayseg

A new segment is named, carries the given trace and span ids, starts in progress, and records start time truncated to milliseconds. A child copies the trace id, uses a new 16-hex-character id, marks itself a subsegment, points at the parent, shares the collector connection, and starts in progress. Recording an error appends an exception whose message is the causal message when present and whose stack is filled only for errors that expose a stack trace (capped at 100 frames). Error is set unless fault or throttle is already set. Each ancestor that has no cause yet stores the child's id as the cause. Capturing a name runs the function inside a child that is submitted in progress and then closed. Closing stamps end time, clears in progress, and writes one UDP datagram: a fixed JSON header line plus the segment document. Submitting in progress writes that datagram at most once, and only while still in progress; a later close overwrites with the finished document.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestSegment_NewSubsegment` | a child has a 16-hex id, copied name/parent/trace, and is in progress with a start time no earlier than the parent |
| `TestSegment_SubmitInProgress` | the second in-progress submit is ignored; close sends a finished document that replaces the pending one |
