# Why hard — grpcxray

predicted_flip: L4

Four interceptor shapes plus a mutex-guarded stream wrapper whose EOF-vs-error rule and single-close flag are easy to miss. Server path must no-op without trace IDs, still connect at construction, and submit in-progress before the handler. Mid-tier models reconstruct unary server from the contract but fail the stream-client close protocol until stubs/tests are visible.
