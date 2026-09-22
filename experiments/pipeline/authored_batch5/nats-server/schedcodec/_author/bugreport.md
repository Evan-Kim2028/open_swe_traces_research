# Bug report — schedcodec

**Title:** Schedule snapshot decode accepts corrupt buffers; `@every`
rejects valid intervals below one minute; `@at` silently ignored.

**Symptoms:**
- On restart, a truncated `schedules.bin` decodes without error,
  leaving `fs.scheduling` partially populated — scheduled messages
  silently vanish or fire at wrong times instead of the file being
  discarded.
- A publisher using `Nats-Schedule: @every 500ms` gets the schedule
  rejected even though the value parses; a `Nats-Schedule-Timezone`
  header is accepted alongside `@at`/`@every` and silently warps the
  fire time.
- `decode` writes entries straight into `schedules` without touching
  `seqToSubj`/`ttls`, so recovered schedules never fire.

**Reproduction:** store a scheduled message, restart the server after
truncating the snapshot mid-entry, then publish with
`@every 500ms` — first should warn+discard, second should be invalid;
both misbehave.

**Expected:** strict version/length/varint validation per entry,
ttl-wheel rebuild on decode, `@at`/`@every` rejecting a time zone, and
`@every` requiring ≥1s.
