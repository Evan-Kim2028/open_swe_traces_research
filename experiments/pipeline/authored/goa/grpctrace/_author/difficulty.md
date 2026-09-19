# Why hard — grpctrace

predicted_flip: L3

Continue-vs-start-vs-discard-vs-zero-rate matrix, stream context wrapping, and client metadata (current span becomes parent). Option wrappers are trivial; the sampler+discard interaction is not. L2 covers the matrix the tests already encode, so a careful mid-tier pass is possible at L2 and likely by L3.
