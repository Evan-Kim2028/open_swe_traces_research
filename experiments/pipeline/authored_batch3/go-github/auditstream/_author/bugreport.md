# Bug report — auditstream

Audit-log streaming configs come back mangled: the vendor-specific settings we
pass in vanish, and the stream-type tag is empty for every vendor, so the
Enterprise audit-log API rejects the serialized configs.

Expected: each constructor returns a config carrying the enabled flag, the
vendor's stream-type tag, and the vendor config we supplied.

Got: only the enabled flag survives; stream type and vendor config are dropped.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
