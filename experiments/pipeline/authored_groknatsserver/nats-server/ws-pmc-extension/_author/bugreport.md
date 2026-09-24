Websocket permessage-deflate negotiation is dead.

A client that sends `Sec-WebSocket-Extensions: permessage-deflate` used to be accepted for compression. The same header with `server_no_context_takeover; client_no_context_takeover` used to be accepted as "both sides drop context". Handshake still completes, but the server never enables compression and never echoes the extension, so large frames are sent uncompressed.
