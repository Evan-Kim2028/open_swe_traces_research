# API left after excision

```go
func validateTimesAt(claims *jwt.UserClaims, now time.Time) (bool, time.Duration)
func validateTimeRangeAt(start, end, now time.Time) (bool, time.Duration)
```

`validateTimes` still exists and forwards to `validateTimesAt(claims, time.Now())`.

Both return `(allowed, remaining)`. `remaining` is how long `now` stays inside the matching window.
