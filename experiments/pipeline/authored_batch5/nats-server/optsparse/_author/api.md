# Exported API — optsparse

Package `server` (module `example.internal/msgkit/v2`) — scalar config
value parsers: the leaf decoders that `ProcessConfigFile` invokes for
durations, host:port, URLs, storage sizes, compression modes, and enum
policies.

Surface (opts.go):
- `parseDuration(field string, tk token, v any, errors, warnings *[]error)
  time.Duration` — duration strings, plus legacy bare-int seconds.
- `parseWriteDeadlinePolicy(tk token, v string, errors *[]error)
  WriteTimeoutPolicy` — default/close/retry.
- `trackExplicitVal(pm *map[string]bool, name string, val bool)` — record
  an explicitly-set boolean.
- `parseListen(v any) (*hostPort, error)` — `int` → port-only,
  `host:port` string → split+validate.
- `parseURLs(a []any, typ string, warnings *[]error) ([]*url.URL,
  []error)` — URL list with duplicate warnings.
- `parseURL(u string, typ string) (*url.URL, error)` — trimmed url.Parse.
- `getStorageSize(v any) (int64, error)` — int64 passthrough or
  `<num><K|M|G|T>` suffix (powers of 2).
- `parseCompression(c *CompressionOpts, chosenModeForOn, tk, mk, mv)
  error` — bool/string/map compression spec incl. rtt thresholds.

Callers: `parseCluster`, `parseGateway`, `parseLeafNodes`,
`parseJetStreamLimits`, `parseAuthorization`, `ProcessConfigFile`
top-level field dispatch. `unwrapValue`, `convertPanicToError(List)`,
`token`, `configErr`/`configWarningErr`, `hostPort`, `CompressionOpts`
stay visible.
