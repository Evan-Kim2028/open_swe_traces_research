# Contract (L2) — xrayseg

A new segment is named, carries the given trace and span ids, starts in progress, and records start time truncated to milliseconds. A child copies the trace id, uses a new 16-hex-character id, marks itself a subsegment, points at the parent, shares the collector connection, and starts in progress. Recording an error appends an exception whose message is the causal message when present and whose stack is filled only for errors that expose a stack trace (capped at 100 frames). Error is set unless fault or throttle is already set. Each ancestor that has no cause yet stores the child's id as the cause. Capturing a name runs the function inside a child that is submitted in progress and then closed. Closing stamps end time, clears in progress, and writes one UDP datagram: a fixed JSON header line plus the segment document. Submitting in progress writes that datagram at most once, and only while still in progress; a later close overwrites with the finished document.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestSegment_NewSubsegment` | a child has a 16-hex id, copied name/parent/trace, and is in progress with a start time no earlier than the parent |
| `TestSegment_SubmitInProgress` | the second in-progress submit is ignored; close sends a finished document that replaces the pending one |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./middleware/xray/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestXraysegNewSegmentProperty`, `TestXraysegStartTimeRandom`, `TestXraysegNewSubsegmentProperty`, `TestXraysegSubsegmentRandom`, `TestXraysegRecordErrorProperty`, `TestXraysegRecordErrorRandom`, `TestXraysegCaptureProperty`, `TestXraysegSubmitInProgressProperty`, `TestXraysegSubmitRandom`, `TestXraysegUDPHeaderProperty`, `TestXraysegAnnotationProperty`: TestXraysegNewSegmentProperty

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
