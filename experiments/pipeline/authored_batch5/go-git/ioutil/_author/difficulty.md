# Difficulty — ioutil

predicted_flip: L2
details: 12

Missed edges: peek-one-byte non-empty check; writer goroutine +
in-flight drain on cancel; panic-to-error conversion; pooled read
window with copy-out ownership; closer fired on ctx.Done;
ReadFinished's two-condition answer; CheckClose first-error-wins;
non-EOF notify filter; offset-tracking ReaderAt adapter; close
chaining order; pooled copy buffer; nop close.

Hardness driver: direct-delegation wrappers pass every happy-path test —
the goroutine machinery exists only for cancellation/panic cases that
tests must set up deliberately.
