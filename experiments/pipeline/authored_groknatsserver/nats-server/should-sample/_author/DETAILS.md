1. A nil `*serviceLatency` returns `(false, nil)`. Inferable: yes
2. `sampling < 0` returns `(false, nil)` without inspecting headers. Inferable: yes
3. `sampling >= 100` returns `(true, nil)` without inspecting headers. Inferable: yes
4. `0 < sampling < 100` samples when `rand.Int32N(100) <= int32(sampling)` (so a configured 1 includes both 0 and 1). Inferable: no
5. If the random branch does not sample, header-driven sampling is still consulted. Inferable: partially
6. `sampling == 0` skips the random branch and goes straight to headers. Inferable: yes
7. Missing or empty parsed headers return `(false, nil)`. Inferable: yes
8. Uber-Trace-Id is four colon-separated fields; the last field is 1 or 2 hex digits. Sampling is on iff the decoded byte has the low bit set. Inferable: no
9. A sampled Uber header returns `newUberHeader(h, tId)`. Inferable: yes
10. `X-B3-Sampled` of `"1"` samples and returns `newB3Header(h)`. Inferable: yes
11. `X-B3-Sampled` of `"0"` denies even if a B3 trace id is present. Inferable: yes
12. Presence of `X-B3-TraceId` without a deny samples and returns `newB3Header(h)`. Inferable: doc
13. Single-field B3 value `"0"` denies. Inferable: no
14. Multi-field B3 (`-` separated) denies when the third field is `"0"`; any other third field (including `"d"`) samples. Inferable: no
15. A sampled combined B3 header is returned as `http.Header{trcB3: b3}` (not the split B3 headers). Inferable: no
16. W3C `traceparent` samples when it has four `-` fields, the flags field is two characters, and the parsed hex value has bit 0x1 set. Inferable: no
17. A sampled W3C header returns `newTraceCtxHeader(h, tId)`. Inferable: yes
18. Header checks run in order Uber, then X-B3-Sampled/TraceId, then combined B3, then traceparent, and the first decision wins. Inferable: yes
