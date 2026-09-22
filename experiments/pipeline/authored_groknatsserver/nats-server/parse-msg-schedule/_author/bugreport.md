Scheduled messages no longer fire on the patterns we document.

An empty schedule used to be accepted as "no schedule". `@at 2026-01-02T15:04:05Z` used to mean fire once at that RFC3339 instant. `@every 5s` used to repeat every five seconds, but `@every 999ms` used to be rejected as too frequent.

Those three cases are now all treated as invalid, so streams that stamp `Nats-Schedule` with `@every 5s` never enqueue the follow-up, and a blank header is reported as a bad schedule instead of being ignored.
