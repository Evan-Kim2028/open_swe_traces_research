# Bug report

Channel handling is broken: `kops`-style channel names no longer resolve against the default
channel base, `"none"` is treated as a URL instead of disabling the channel, channel YAML fails
to parse, recommended/required upgrades never fire (or fire for equal versions), version-range
specs are not matched, `FindImage` ignores architecture and version filters, and package versions
cannot be looked up.

Expected: `ResolveChannel("none")` returns nil; relative names resolve under the default channel
base; recommendations fire only for strictly newer versions; the first matching `Range` spec wins;
images filter on provider, architecture and version range; upstream image prefixes are recognised.

Reproduce with:

```
go test -count=1 ./pkg/apis/kops/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
