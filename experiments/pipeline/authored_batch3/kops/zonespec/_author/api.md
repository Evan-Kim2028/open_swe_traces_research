# Exported API — zonespec

Package `dns-controller/pkg/dns` (importable as `example.internal/clustkit/dns-controller/pkg/dns`).

- `func ParseZoneSpec(s string) (*ZoneSpec, error)` — parse one zone selector `name`, `*/id`, or `name/id` into `ZoneSpec{Name, ID}`.
- `func ParseZoneRules(zones []string) (*ZoneRules, error)` — parse a list of selectors into `ZoneRules{Zones, Wildcard}`.
- `func (r *ZoneRules) MatchesExplicitly(zone dnsprovider.Zone) bool` — true when the zone matches a non-wildcard rule.

Production callers: `dns-controller/pkg/dns/dnscontroller.go`, `dns-controller/cmd/dns-controller/main.go` (`--zones` flag).
