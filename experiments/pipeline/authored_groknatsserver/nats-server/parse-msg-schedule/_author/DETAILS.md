1. An empty pattern returns a zero time, `repeating=false`, `ok=true`. Inferable: no
2. A pattern starting with `@at ` is a one-shot RFC3339 timestamp taken from the remainder after that prefix. Inferable: doc
3. `@at` is illegal when `loc` is non-nil. Inferable: doc
4. `@at` parse failure returns zero time, `repeating=false`, `ok=false`. Inferable: yes
5. A pattern starting with `@every ` is a repeating `time.ParseDuration` interval taken from the remainder after that prefix. Inferable: doc
6. `@every` is illegal when `loc` is non-nil. Inferable: doc
7. `@every` intervals shorter than one second are illegal. Inferable: no
8. `@every` computes `time.Unix(0, ts).UTC().Round(time.Second).Add(dur)`; if that instant is already in the past it becomes `time.Now().UTC().Round(time.Second).Add(dur)`. Inferable: doc
9. The alias `@yearly` (and `@annually`) is a repeating cron schedule that fires at the first second of 1 January. Inferable: no
10. The alias `@monthly` is a repeating cron schedule that fires at the first second of the first day of each month. Inferable: no
11. The alias `@weekly` is a repeating cron schedule that fires at the first second of Sunday. Inferable: no
12. The aliases `@daily` and `@midnight` are repeating cron schedules that fire at the first second of each day. Inferable: no
13. The alias `@hourly` is a repeating cron schedule that fires at the first second of each hour. Inferable: no
14. Any other pattern is handed to `parseCron`; a parse error is illegal (`ok=false`). Inferable: yes
15. A cron next-time that is already in the past is recomputed from `time.Now().UTC().Round(time.Second)` as the new `ts`. Inferable: doc
16. Successful `@every` and cron results have `repeating=true`. Inferable: yes
