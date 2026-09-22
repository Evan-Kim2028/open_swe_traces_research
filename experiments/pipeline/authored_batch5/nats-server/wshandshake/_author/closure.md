# Closure — wshandshake

Package: `server`. File: `websocket.go`.

Removed (6 funcs stubbed): `wsAcceptKey`, `wsHeaderContains`,
`wsPMCExtensionSupport`, `srvWebsocket.checkOrigin`, `wsGetHostAndPort`,
`wsMakeChallengeKey`.

Kept: `wsUpgrade` (the caller), `wsUpgradeResult`, `srvWebsocket` fields
(`sameOrigin`, `allowedOrigins`, `allowedOrigin{scheme,port}`), `wsGUID`,
`wsPMCExtension`/`wsPMCSrvNoCtx`/`wsPMCCliNoCtx` constants,
`wsReturnHTTPError`, `wsSetOriginOptions`, frame codec and read loop.
Imports `crypto/fips140`, `crypto/sha1` blanked.

Tests snipped in `server/websocket_test.go`: `TestWSCheckOrigin`,
`TestWSUpgradeValidationErrors`, `TestWSCompressNegotiation`.
