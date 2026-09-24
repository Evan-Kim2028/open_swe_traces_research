# Exported API — mondecode

Package `server` (module `example.internal/msgkit/v2`) — HTTP monitoring
request decoding and display helpers used by the `/connz`, `/subsz`,
`/routez`, `/gatewayz`, `/leafz`, `/accountz` handlers.

Surface (monitor.go):
- `decodeBool(w, r, param) (bool, error)` — query param → bool; empty →
  false; parse failure writes 400 + message and returns the error.
- `decodeUint64(w, r, param) (uint64, error)` — base-10 u64.
- `decodeInt(w, r, param) (int, error)` — Atoi.
- `decodeState(w, r) (ConnState, error)` — `state` param: open/closed/
  any/all (case-insensitive); empty → ConnOpen; unknown → 400 + error.
- `decodeSubs(w, r) (subs, subsDet bool, err error)` — `subs=detail` →
  subsDet; else bool-decoded subs.
- `myUptime(d time.Duration) string` — compact `22y32d4h4m22s` rendering
  dropping leading zero units.
- `redactBearerJWT(userJWT string) string` — empty when the JWT decodes to
  a bearer-token user claim.

Callers: all `Handle*z` HTTP endpoints and `Connz`/`Subsz`/etc. option
parsing, `server.go` uptime logging. `ConnState` constants, `jwt`
package stay visible.
