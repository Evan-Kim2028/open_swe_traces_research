# Closure — schedcodec

File: `server/scheduler.go`

Stubbed (3 symbols):

- `MsgScheduling.encode` — snapshot writer (version+count+stamp
  header, per-entry subject/ts/seq)
- `MsgScheduling.decode` — snapshot reader with strict truncation
  checking
- `parseMsgSchedule` — schedule-pattern dispatcher (`@at`, `@every`,
  `@yearly`…aliases, cron fallback)

Retained as scaffolding: `parseCron` + all of cron.go (banked in
`cronparse`), `MsgScheduling` add/init/update/remove/resetTimer/
getScheduledMessages, `thw` wheel, `headerLen`,
`ErrMsgScheduleInvalidVersion`. Imports blanked after excision:
`encoding/binary`, `io`, `strings`.

Test coverage snipped (restored for gold/cheat):
`TestFileStoreMessageScheduleEncodeDecode` (100k-entry round trip +
ttl-wheel equality) and `TestFileStoreMessageScheduleDecodeRejectsMalformed`
(6-case truncation table) in filestore_test.go.
