# Exported API — distros

Package `util/pkg/distributions` (importable as `example.internal/clustkit/util/pkg/distributions`).

- `type Distribution{packageFormat, project, id string; version float32}` — OS identity; instances exposed as `DistributionDebian12`, `DistributionUbuntu2404`, `DistributionRhel9`, `DistributionFlatcar`, etc.
- Predicates: `IsDebianFamily`, `IsDebian`, `IsUbuntu`, `IsAmazonLinux`, `IsRHELFamily`, `HasDNF`, `IsSystemd`, `HasLoopbackEtcResolvConf`, `ForceNftables`.
- `DefaultUsers() ([]string, error)`, `Version() float32`.

Production callers: `nodeup/pkg/model/*` (package selection, kube-proxy mode, OS users), `upup/pkg/fi/cloudup/*` distro detection paths.
