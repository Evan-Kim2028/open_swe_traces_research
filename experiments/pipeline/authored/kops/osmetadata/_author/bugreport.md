# Bug report

OpenStack node-local metadata lookup panics. Config-drive vs link-local HTTP fallback no longer respects search order, and when every source fails the error from the last attempt is lost (or the first error is returned instead).

Reproduce with:

```
go test -count=1 ./upup/pkg/fi/cloudup/openstack/openstackmetadata/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
