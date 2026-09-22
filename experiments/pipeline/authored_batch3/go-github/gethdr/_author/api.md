# Exported API — gethdr

`HookRequest.GetHeader` and `HookResponse.GetHeader` expose HTTP header
lookup on webhook deliveries, so callers can pull `X-GitHub-*` headers
without knowing the exact case the transport recorded them in.

- `getHeader` (unexported, excised) — shared lookup behind both methods.
