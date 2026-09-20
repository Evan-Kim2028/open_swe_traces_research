# Closure — distros

Package: `util/pkg/distributions` (`example.internal/clustkit/util/pkg/distributions`).

Files: `util/pkg/distributions/distributions.go` (11 methods on `Distribution`).

Removed functions (bodies stubbed): `IsDebianFamily`, `IsDebian`, `IsUbuntu`, `IsAmazonLinux`, `IsRHELFamily`, `HasDNF`, `IsSystemd`, `DefaultUsers`, `HasLoopbackEtcResolvConf`, `Version`, `ForceNftables`.

Exported entry point(s): the `Distribution` predicate methods — consumed by nodeup/package builders throughout the tree; the `Distribution*` package vars remain intact so every caller still compiles.
