# Difficulty — reflectfmt

predicted_flip: L2
details: 12
Commitments: root-first visit order with nil field, SkipReflection pruning semantics, unexported-field skipping, JSONNames tag handling (incl. `json:"-"`), MapKey/ArrayIndex path elements, nil-pointer visit-once rule, DeprecatedDoubleVisit, the unusual non-primitive string/slice set, quoted-string/null formatting, compositional type names, fatal-on-marshal-error, and shallow type-assertion error detection.

Hardness driver: the visit protocol has three interacting rules (visitor on root, nil-but-visited pointers, double-visit mode); `json:"-"` producing the literal name `-` is adversarial; and `IsPrimitiveValue` reads opposite to its name for the two most common kinds.
