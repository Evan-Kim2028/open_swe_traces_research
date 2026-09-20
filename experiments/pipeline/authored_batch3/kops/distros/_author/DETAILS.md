# Details — distros

1. `IsDebianFamily` tests `packageFormat == "deb"` and `IsRHELFamily` tests `packageFormat == "rpm"`; immutable distros (flatcar, containeros) carry an empty packageFormat so both are false. Inferable: yes — the field is documented on the struct.
2. `IsDebian`/`IsUbuntu`/`IsAmazonLinux` test the `project` field exactly — Ubuntu is debian-family but `IsDebian` is false for it. Inferable: yes — doc comments state it.
3. `HasDNF` is false for non-rpm distros; per-project version gates apply (rhel/rocky/centos >= 8, fedora >= 22, amazonlinux always); an UNKNOWN rpm project returns true with a warning. Inferable: no — the gates and the unknown→true default are arbitrary.
4. `IsSystemd` always returns true — every supported distro is systemd. Inferable: no — a constant predicate.
5. `DefaultUsers` returns a per-project list: debian→[admin,root], ubuntu→[ubuntu,root], centos→[centos], rhel & amazonlinux→[ec2-user], rocky→[rocky], flatcar→[core]; unknown projects error. Inferable: no — the user lists are arbitrary facts.
6. `HasLoopbackEtcResolvConf` is true for ubuntu and flatcar; otherwise it stat()s `/run/systemd/resolve/resolv.conf` on the HOST — the result depends on the test environment, not the distro. Inferable: no.
7. `Version` returns the project-scoped float (22.04 for jammy, 9 for rhel9). Inferable: yes.
8. `ForceNftables` is rpm-family-only MINUS an allowlist (amzn2023/2027, rhel8/9, rocky8/9, centos9) — fedora and all 10+ rhel-family releases force nftables; comparison is by value. Inferable: no — the allowlist is an arbitrary table.
