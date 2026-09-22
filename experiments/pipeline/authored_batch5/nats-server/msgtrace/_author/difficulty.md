# difficulty — msgtrace

Medium. genHeaderMapIfTraceHeadersPresent is a fussy line parser: key to
first colon, both-side space/tab value trim, empty-key/value drop,
stop-on-malformed, case-sensitivity asymmetry between the two special
keys, the 4-token traceparent + sampled-bit rule, and the
external-only bool. Easy to write a version that passes the obvious cases
and fails the malformed/prefix/suffix table rows.
