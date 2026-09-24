# Details — firesources

1. `ResourcesMatch` reads 8192-byte chunks with `io.ReadFull`; EOF/ErrUnexpectedEOF terminate
   the loop — the final comparison uses only the bytes actually read. Different lengths →
   false, not error. Inferable: yes — chunked compare is required to stream.
2. `CopyResource`/`FileResource.Open`/`VFSResource.Open` return `os.IsNotExist` errors
   UNWRAPPED so callers can distinguish missing vs unreadable; other open errors are wrapped.
   Inferable: partially — the unwrap rule is a deliberate contract.
3. `BytesResource.MarshalJSON` marshals `string(b.data)` — the bytes appear as a JSON string,
   not base64. Inferable: no — string vs base64 is arbitrary.
4. `TaskDependentResource.Open` errors when `Resource` is nil; `IsReady` is `Resource != nil`.
   Inferable: yes.
5. `TaskDependentResource.GetDependencies` returns exactly the producing task (singleton).
   Inferable: yes.
6. `functionResource.Open` memoizes: `fn` runs on first call only; a nil cached slice means
   "not yet run", so a `fn` returning empty bytes re-runs on each call (only non-nil caches).
   Inferable: partially — nil-vs-empty caching edge is an implementation detail.
7. `StringResource`/`BytesResource` `Open` never error. Inferable: yes.
