TLS-mapped cluster and gateway auth no longer accepts DNS SANs.

A server cert with SAN `connect.Y.local` used to match a route URL `nats://connect.Y.local`. `Foo.Bar` used to match `foo.Bar` (case did not matter). `Y-X-red-mgmt` used to reject `X-X-red-mgmt`.

After the last change every SAN check fails, so `tls_map` / `tls_check_known_urls` never authorizes a route or gateway even when the certificate names the peer.
