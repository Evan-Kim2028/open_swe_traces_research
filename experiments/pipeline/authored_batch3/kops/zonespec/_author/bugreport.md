# Bug report

DNS zone allow-list rules select the wrong zones: named rules fail to match zones they should admit, id-based rules match nothing or everything, wildcard handling admits no zones at all (or all of them), and a rule carrying both a name and an id accepts zones that only satisfy one half.

Expected: a rule `example.com` admits a zone named `example.com`; a rule `*/1234` admits any zone whose id is `1234` regardless of name; `name/id` requires both; `*` or an empty rule list admits everything.

Reproduce with:

```
go test -count=1 ./dns-controller/pkg/dns/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
