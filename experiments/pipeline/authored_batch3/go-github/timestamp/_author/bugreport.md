# Bug report — timestamp

Some API payloads carry times as large bare JSON numbers. Most numeric values
decode fine, but a range of large Unix-second timestamps — roughly the ones
encoding dates between the late 2200s and the early 3000s — get misread as
millisecond values and come out as 1970s dates instead of the encoded year.

Expected: every numeric time field decodes to the instant it encodes, whatever
its magnitude.

Got: a band of large numeric timestamps decodes to dates around 1970.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
