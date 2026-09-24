# Details — fiutils

1. `StringSlicesEqual` is order-sensitive; `StringSlicesEqualIgnoreOrder` sorts a copy.
   `["a","b"]` vs `["b","a]` → false / true. Inferable: yes.
2. `HashString` returns lowercase sha256 hex (64 chars). Inferable: yes.
3. `IsIPv6IP`/`IsIPv6CIDR` use the `To4() == nil` test — `"::ffff:10.0.0.1"` (v4-mapped) is
   NOT IPv6. `IsIPv4CIDR` additionally rejects strings containing `":"` — a v6 CIDR that
   parses but is v4-mappable is false. Inferable: partially — the extra `:` check is a choice.
4. `ParseCIDRNotation` accepts only the exact `/N#hex` form (`^/(\d+)#([a-f0-9]+)$`) —
   `/20#1`, not `"20#1"` or `/20#0x1`. Inferable: no — the micro-format is arbitrary.
5. `CIDRSubnet` uses `cidr.SubnetBig` with `newSize - oldSize` additional bits — `netNum`
   is the subnet index, not an offset. Inferable: partially.
6. `SanitizeString` maps disallowed runes to `_` (not removed) and truncates to the LAST
   200 chars (a long prefix is dropped, not the suffix). Inferable: no — allowed charset,
   replacement char and tail-truncation are all arbitrary.
7. `ExpandPath` expands ONLY a `~/` prefix (bare `~` is untouched) via `homedir.HomeDir()`.
   Inferable: partially.
8. `YamlUnmarshal`/`YamlMarshal` wrap `sigs.k8s.io/yaml` — JSON field tags apply.
   Inferable: yes.
