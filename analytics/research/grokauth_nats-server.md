# grokauth nats-server — 10 units

All units live under `units/<name>/_author/`. `src/` was not modified. Line ranges do not overlap claimed cuts. Ten files, ten behaviours.

| name | file | what was cut | why not recallable | Inferable (yes/doc/partially/no) | cheat/gold |
|---|---|---|---|---|---|
| index-placeholders | server/subject_transform.go | `indexPlaceHolders` — `$N` and mustache mapping-function parser | House language: per-function arities, `$` non-int fallback, split-delimiter bans, int32 caps | 12 / 3 / 2 / 1 | 0.140 |
| parse-msg-schedule | server/scheduler.go | `parseMsgSchedule` — `@at` / `@every` / cron aliases | Three dialects plus loc-forbidden-on-@at/@every, 1s floor, skip-ahead | 3 / 6 / 0 / 7 | 0.333 |
| jwt-path-for-key | server/dirstore.go | `(*DirJWTStore).pathForKey` — JWT file path | Shard dir is the **last two** nkey chars, not a hash or prefix | 5 / 0 / 0 / 2 | 0.580 |
| jwt-time-windows | server/jwt.go | `validateTimesAt` + `validateTimeRangeAt` | Exclusive endpoints, `15:04:05` clocks, midnight wrap by +1 day, max remaining | 7 / 4 / 1 / 2 | 0.258 |
| encode-consumer-state | server/store.go | `encodeConsumerState` | Magic 22 / version 2, floor-delta pending, inverted second timestamps | 3 / 2 / 3 / 3 | 0.137 |
| get-storage-size | server/opts.go | `getStorageSize` | Uppercase K/M/G/T as `1<<10/20/30/40`; empty string is 0 | 6 / 0 / 4 / 2 | 0.358 |
| write-leaf-sub | server/leafnode.go | `(*client).writeLeafSub` | `LS+`/`LS-` plus "space in key ⇒ append queue count" | 6 / 2 / 0 / 0 | 0.467 |
| should-sample | server/accounts.go | `shouldSample` | Uber/B3/W3C flag packing, `"d"` is not a deny, `rand <= sampling` includes 0 | 10 / 1 / 1 / 6 | 0.293 |
| ws-pmc-extension | server/websocket.go | `wsPMCExtensionSupport` | `checkPMCOnly` short-circuit; takeover params only after the matching token | 9 / 0 / 1 / 1 | 0.490 |
| dns-alt-name-matches | server/auth.go | `dnsAltNameMatches` | Equal label count; `*` only leftmost; RFC 6125 "may" treated as will-not | 5 / 4 / 1 / 0 | 0.445 |

Cheat patches special-case only the worked examples in each `bugreport.md`. Ratios are byte sizes of `cheat.patch` / `gold.patch`.
