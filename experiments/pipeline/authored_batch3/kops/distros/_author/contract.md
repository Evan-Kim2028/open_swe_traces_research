# Contract (L2) — distros

`Distribution` predicates classify OS images by `packageFormat` (deb/rpm/immutable) and `project`. Debian-family tests the format; `IsDebian`/`IsUbuntu`/`IsAmazonLinux` test the project exactly. `HasDNF` gates on rpm-family plus per-project version floors, defaulting true for unknown rpm projects. `IsSystemd` is always true. `DefaultUsers` is a per-project user table; unknown projects error. `HasLoopbackEtcResolvConf` is true for ubuntu/flatcar else probes the host filesystem. `ForceNftables` is rpm-family minus an explicit working-iptables allowlist. `Version` returns the project-scoped float.

## Coverage of original in-tree tests

No in-tree tests were deleted for this unit — `identify.go` never invokes the predicates, and the package carries no `*_test.go` exercising them. Reachability is via `nodeup/pkg/model` consumers (package selection, kube-proxy mode, OS user creation).
