# Exported API — wshandshake

Package `server` (module `example.internal/msgkit/v2`) — WebSocket upgrade
validation: handshake key derivation, header token matching, permessage-deflate
negotiation, and origin checking.

Surface: `wsAcceptKey(key string) string` (Sec-WebSocket-Accept),
`wsMakeChallengeKey() (string, error)` (Sec-WebSocket-Key for outbound
leafnode dials), `wsHeaderContains(h http.Header, name, value string) bool`
(comma-separated token match), `wsPMCExtensionSupport(h http.Header,
checkPMCOnly bool) (bool, bool)` (extension + no-context-takeover flags),
`(w *srvWebsocket) checkOrigin(r *http.Request) error`,
`wsGetHostAndPort(tls bool, hostport string) (string, string, error)`.

Callers: `wsUpgrade`, leafnode websocket dialer (`leafnode.go`),
`wsSetOriginOptions` configures the checked state.
