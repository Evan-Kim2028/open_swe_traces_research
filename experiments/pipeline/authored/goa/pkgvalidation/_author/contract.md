# Contract (L2) — pkgvalidation

A format check returns no error when the string matches the named format and an invalid-format error (including the value, format name, and underlying parse error) when it does not. Unknown format names are rejected. Date is a full-date, date-time is RFC3339, email is RFC5322, hostname matches the hostname regex, URI is a request URI, MAC/CIDR/regexp/JSON/RFC1123 use the standard library parsers. IPv4 must be a dotted quad that parses as IP; IPv6 must parse as IP and must not be a dotted quad; "ip" accepts either. UUID accepts the hyphenated, 32-hex, braced, and urn:uuid renderings of an RFC4122 UUID and rejects other variants and malformed strings. A pattern check compiles each regular expression once and reuses it; a non-match is an invalid-pattern error naming the value and pattern.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestValidateFormat` | valid and invalid examples for every supported format, including UUID renderings and IPv4/IPv6 cross-rejection |
| `TestValidatePattern` | a matching value passes; a non-matching value is an invalid-pattern error |
