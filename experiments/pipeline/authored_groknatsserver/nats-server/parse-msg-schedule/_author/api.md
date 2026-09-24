# API left after excision

```go
func parseMsgSchedule(pattern string, loc *time.Location, ts int64) (time.Time, bool, bool)
```

Returns `(next, repeating, ok)`.

- `next` is the next fire time
- `repeating` is true for interval and cron forms, false for one-shot `@at`
- `ok` is false when the pattern is illegal

`parseCron` remains and is the backend for five-or-six-field cron text.
