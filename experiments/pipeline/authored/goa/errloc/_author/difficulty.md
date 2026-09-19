# Why hard — errloc

predicted_flip: L5

Call-stack heuristics under inlining: skip this module by relative path from the current source file, skip registered design packages after stripping `@version` cache segments, keep `_test.go` frames, start scanning two callers up, and format validation errors from a func-pointer FileLine rather than the stack. A full prose contract still underspecifies skip order and path normalization; models typically need a hidden test file (L5) before the locations stabilize.
