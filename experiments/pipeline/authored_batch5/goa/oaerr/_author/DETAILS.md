# Commitments — oaerr

1. Content-type precedence: response setting wins, then the result type's `ContentType`, then `application/json`. In-tree coverage: v2/v3 builder tests (trimmed). Inferable: doc — ResponseContentType's comment states the order.
2. `ErrorResponseExample` returns `(nil, false)` for non-error results, when authored examples exist, or when no generated example is available. In-tree coverage: description-ownership tests (trimmed). Inferable: doc — its comment.
3. Flag fields (name/temporary/timeout/fault) are only written onto keys already present in the example — fields mapped to headers or cookies are not re-added to the body. In-tree coverage: `TestSharedErrorResponseDescriptions` et al (trimmed). Inferable: doc — setExampleField's comment.
4. A non-object example is returned unchanged with `true`. In-tree coverage: example tests (trimmed). Inferable: partially.
