# Closure — hdrsurgery

Package `server`, file `server/client.go`.

Removed (stubbed): `sliceHeader`, `getHeader`, `getHeaderKeyIndex`,
`removeHeaderStatusIfPresent`, `removeHeaderIfPresent`,
`removeHeaderIfPrefixPresent`, `isServiceReply`, `isJSAckSubject`,
`jsAckDeliverIdx`, `replyHasJSAckSuffix`, `isReservedReply`,
`splitSubjectQueue`, `splitArg`.

Retained: `genHeader`, `setHeader`, `hasGWRoutedReplyPrefix` (gateway.go),
`IsValidSubject`, all constants (`replyPrefix`, `jsAckPre`, `jsAckPreLen`,
`emptyHdrLine`, `hdrLine`, `CR_LF`, `LEN_CR_LF`, `MAX_MSG_ARGS`), the
entire client read/write machinery.

No import changes (all imports still used elsewhere in the file).

Tests snipped in server/client_test.go: TestSliceHeader,
TestSliceHeaderOrderingPrefix, TestRemoveHeaderIfPrefixPresent,
TestRemoveHeaderIfPrefixPresentSkipsValueMatches,
TestRemoveHeaderIfPresentSkipsValueMatches,
TestRemoveHeaderIfPresentDuplicates,
TestRemoveHeaderIfPresentOrderingPrefix, TestReplyHasJSAckSuffix,
TestSplitSubjectQueue. No test files deleted. `replyHasJSAckSuffix` is
also exercised by a jetstream test left in place.
