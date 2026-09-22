1. The function reads the `Sec-Websocket-Extensions` header (Go's canonical MIME key). Inferable: yes
2. Each header value is split on commas into extensions; each extension is split on semicolons into a token plus parameters. Inferable: yes
3. Tokens and parameters are trimmed of space and tab. Inferable: yes
4. The extension token is matched case-insensitively against `permessage-deflate`. Inferable: yes
5. When `checkPMCOnly` is true, a match returns `(true, false)` without inspecting parameters. Inferable: yes
6. When `checkPMCOnly` is false, only parameters *after* the matching token in that extension are examined. Inferable: no
7. `server_no_context_takeover` and `client_no_context_takeover` are matched case-insensitively. Inferable: yes
8. If both of those parameters appear, the result is `(true, true)`. Inferable: yes
9. If the extension is present but one or both parameters are missing, the result is `(true, false)`. Inferable: yes
10. If no extension matches, the result is `(false, false)`. Inferable: yes
11. The first matching `permessage-deflate` extension decides; later list entries are not consulted once that extension has been fully scanned. Inferable: partially
