# API — schedcodec

Module: `example.internal/msgkit/v2`, package `server`, file `server/scheduler.go`.

Excised symbols:

- `func (ms *MsgScheduling) encode(highSeq uint64) []byte` — binary
  snapshot of all pending message schedules.
- `func (ms *MsgScheduling) decode(b []byte) (uint64, error)` —
  restores the snapshot, returning the stamped high sequence.
- `func parseMsgSchedule(pattern string, loc *time.Location, ts int64)
  (time.Time, bool, bool)` — turns a schedule pattern (`@at`,
  `@every`, predefined `@daily`/`@hourly`/…, or a 6-field cron spec)
  into the next fire time, a repeating flag, and a validity flag.

Callers: filestore/memstore persist `ms.encode` on shutdown and call
`ms.decode` during recovery; `nextMessageSchedule`/`getMessageSchedule`
(stream.go) call `parseMsgSchedule` when a published message carries
the `Nats-Schedule` header.

Retained dependencies (visible in excised tree): `parseCron`
(cron.go — banked separately), `MsgScheduling` map/set bookkeeping,
`thw` timer wheel, `headerLen`, `ErrMsgScheduleInvalidVersion`.
