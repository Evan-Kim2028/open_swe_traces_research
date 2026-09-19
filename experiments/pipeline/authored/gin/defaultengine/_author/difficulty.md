# Why hard — defaultengine

predicted_flip: L3
25 trivial wrappers around one subtle state invariant — a lazily-initialized shared engine. A plausible cheat (fresh engine per call) makes every individual call work but loses cross-call state, which is exactly what the suite checks.
