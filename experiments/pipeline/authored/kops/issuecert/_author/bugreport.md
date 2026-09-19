# Bug report

Certificate issuance panics. Self-signed CAs, client/server certificates, SAN split (DNS vs IP), default ten-year lifetime, and “generate a key if none was provided” are all gone. PEM load of keys and certificates also panics, so any round-trip through PEM fails.

Reproduce with:

```
go test -count=1 ./pkg/pki/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
