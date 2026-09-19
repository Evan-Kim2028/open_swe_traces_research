# Exported API — pkgvalidation

```
func ValidateFormat(name string, val string, f Format) error
func ValidatePattern(name, val, p string) error
```

ValidateFormat checks val against json-schema draft-4 formats (date, date-time, uuid, email, hostname, ipv4/ipv6/ip, uri, mac, cidr, regexp, json, rfc1123). Failures wrap InvalidFormatError. UUID accepts RFC4122 hyphenated, raw, braced, and urn:uuid forms and rejects non-RFC4122 variants. IPv4/IPv6 additionally use a dotted-quad regex so IPv6 loopback is not valid IPv4 and vice versa. ValidatePattern compiles each pattern once (mutex-protected map) and wraps InvalidPatternError on mismatch.

## Pre-existing callers

Generated Validate methods on payloads/results; package tests.
