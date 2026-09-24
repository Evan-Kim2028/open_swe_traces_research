# Bug report

Operating-system detection is wrong: debian-family systems are misclassified, RPM-based distributions report the wrong package tooling, the default SSH users list comes back wrong or empty for several distributions, and kube-proxy mode selection treats distributions that still have working iptables as if they required nftables.

Expected: an Ubuntu image is debian-family but is not Debian; an Amazon Linux image uses dnf; the default user for a RHEL-family image is `ec2-user`; a distribution with working iptables does not force nftables proxy mode.

Reproduce with:

```
go test -count=1 ./util/pkg/distributions/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
