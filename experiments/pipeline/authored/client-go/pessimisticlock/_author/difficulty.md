# Why hard — pessimisticlock

two parallel response protocols (normal vs force/only-if-exists), wait/killed/timeout semantics with deadlock interaction, per-key existence/value plumbing, and region-error surgical retry. ~500 lines of response-classification logic.
