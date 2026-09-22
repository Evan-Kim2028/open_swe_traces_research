# Bug report — hdrsurgery

The NATS header surgery helpers and reply-subject classifiers are missing.
Every call to `getHeader`, `sliceHeader`, `getHeaderKeyIndex`,
`removeHeaderStatusIfPresent`, `removeHeaderIfPresent`,
`removeHeaderIfPrefixPresent`, `isServiceReply`, `isJSAckSubject`,
`jsAckDeliverIdx`, `replyHasJSAckSuffix`, `isReservedReply`,
`splitSubjectQueue`, or `splitArg` panics. Header-aware message paths
(JS expected-* headers, ClientInfoHdr, status stripping on republish) and
SUB arg tokenization are broken.

Reproduce:

    go test ./server/ -run 'TestSliceHeader|TestRemoveHeaderIf|TestReplyHasJSAckSuffix|TestSplitSubjectQueue'

Restore the helpers with their real semantics: CRLF-anchored key lookup
that survives prefix-shadowing keys, capacity-limited borrowing in
sliceHeader, removal loops that handle duplicates and collapse to nil at
the empty header, the 8-dots-before-@ rule for JS ack delivery suffixes,
and whitespace-run field splitting with subject validation.

Work only from the repository and test output. Do not use web search or
any tool that accesses the internet. This repository is fully
self-contained; do NOT fetch upstream sources. Preserve all unrelated
tests.
