# Details — reqsource

1. `GetRequestSource` returns `"unknown"` when the receiver is nil OR both
   `RequestSourceType` and `ExplicitRequestSourceType` are empty — even if
   `RequestSourceInternal` is set (avoids marking unset requests internal).
   Inferable: partially — documented intent, exact rule is a choice.
2. Otherwise the label joins with `_`: `{internal|external}_{type}` where
   empty `RequestSourceType` falls back to `"unknown"`, plus
   `_{explicitType}` appended only when non-empty AND different from
   `RequestSourceType` — `internal_test_lightning`,
   `external_unknown_lightning`, `external_test` (explicit==type dedups).
   Inferable: partially — layout visible, the dedup rule is not.
3. `IsInternalRequest` is `strings.HasPrefix(source, "internal")` — prefix
   on the bare word, so `"internalx"` counts as internal. Inferable: no.
4. `IsRequestSourceInternal` is nil-safe and defers to the composed label.
   `BuildRequestSource` composes `GetRequestSource` on a fresh struct.
   Inferable: yes.
5. `WithInternalSourceType`/`WithInternalSourceAndTaskType` store a
   `RequestSource` value (not pointer) under `RequestSourceKey`;
   `RequestSourceFromCtx` returns `"unknown"` when absent. Inferable:
   partially.
6. `WithResourceGroupName`/`ResourceGroupNameFromCtx` use a distinct
   unexported key type; missing key yields `""`. Inferable: doc.
