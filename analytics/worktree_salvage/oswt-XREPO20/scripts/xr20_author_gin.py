#!/usr/bin/env python3
"""Write _author artifacts for the 14 new gin->validator xrepo20 units.

Content lives here so the batch is reproducible: run once, artifacts land in
experiments/pipeline/authored_batch2/gin/<family>/_author/. Hidden tests are
copied from /tmp/xrh (verified pristine-passing) — or pass --hidden-src.
"""
from __future__ import annotations

import argparse
import shutil
from pathlib import Path

ROOT = Path("/home/evan/Documents/oswt-XREPO20")
AUTHOR = ROOT / "experiments/pipeline/authored_batch2/gin"
HIDDEN_SRC = Path("/tmp/xrh")

REPRO = """Reproduce with:

```
tests/test.sh
```

(which copies the hidden suite into the tree and runs
`go test -count=1 -timeout 15m ./binding/...`)

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break existing
behavior.
"""

API = """# Exported API — {fam}

Consumer-facing: `binding.Validator` (`StructValidator`), `Context.ShouldBind*`,
`binding:"{tags}"` tag DSL. The hidden suite only touches exported
consumer API; the excised functions are unexported and live in the dependency
module.
"""

def api(fam, tags):
    return API.format(fam=fam, tags=tags)

def bugreport(symptom):
    return f"# Incorrect behavior\n\n{symptom}\n\n{REPRO}"

def contract(title, prose, rows):
    r = "\n".join(f"| `{t}` | {s} |" for t, s in rows)
    return (f"# Contract (L2) — {title}\n\n{prose}\n\n"
            f"## Coverage of hidden tests\n\n| test | contract sentence |\n|---|---|\n{r}\n")

def details(items):
    return "\n".join(f"{i+1}. {d}" for i, d in enumerate(items)) + "\n"

def closure(fam, funcs, kept, reach):
    return (f"# Closure — {fam} (cross-repo)\n\n"
            "- Consumer C: `example.internal/httprouter` (obfuscated gin @5c6a15f)\n"
            "- Library L: `deps/validator` = `github.com/go-playground/validator/v10` v10.30.3, nested module via `replace`.\n"
            f"- Excised funcs: {funcs}\n"
            f"- Kept in place: {kept}\n"
            f"- Reachability: {reach}\n")

def difficulty(fam, body, items=()):
    d = "\n".join(f"  {i+1}. {x}" for i, x in enumerate(items))
    return f"# Why hard — {fam}\n\npredicted_flip: L2\ndetails:\n{d}\n\n{body}\n"


PRISTINE_DV = (ROOT / "experiments/xrepo20/pair/gin/src/binding/default_validator.go").read_text()


def _diff(old: str, new: str, rel: str) -> str:
    import difflib
    return "".join(difflib.unified_diff(
        old.splitlines(keepends=True), new.splitlines(keepends=True),
        fromfile=f"a/{rel}", tofile=f"b/{rel}"))


# cheat patch template: wrap validateStruct, recover, shallow recheck
def gin_cheat(fam, check_body, check_sig="obj any", note="", add_imports=("errors",)):
    """Patch derived from the pristine file so hunk offsets are always right."""
    fn = f"xr{fam.capitalize()}Check"
    old = PRISTINE_DV
    new = old.replace(
        "\tv.lazyinit()\n\treturn v.validate.Struct(obj)",
        f"\treturn v.validateStruct{fam.capitalize()}(obj)",
        1,
    )
    assert new != old, "validateStruct body not found"
    if add_imports:
        imps = "".join(f'\t"{i}"\n' for i in add_imports)
        new = new.replace("import (\n", "import (\n" + imps, 1)
    new += (
        f"\n// xr{fam.capitalize()} fallback: when the engine aborts on {fam} tags we\n"
        f"// re-check them locally. {note}\n"
        f"func (v *defaultValidator) validateStruct{fam.capitalize()}(obj any) (err error) {{\n"
        f"\tdefer func() {{\n"
        f"\t\tif r := recover(); r != nil {{\n"
        f"\t\t\terr = {fn}(obj)\n"
        f"\t\t}}\n"
        f"\t}}()\n"
        f"\tv.lazyinit()\n"
        f"\treturn v.validate.Struct(obj)\n"
        f"}}\n\n"
        f"func {fn}({check_sig}) error {{\n{check_body}\n}}\n"
    )
    return _diff(old, new, "binding/default_validator.go")


# Per-unit content -----------------------------------------------------------

UNITS = {}

UNITS["mailfmt"] = dict(
    tags="email",
    api_tags="email",
    api=api("mailfmt", "email"),
    bug=bugreport(
        "Struct binding crashes on email address rules. When a bound field is\n"
        "annotated with the `email` format rule, validation aborts with a\n"
        "`panic:` instead of accepting valid addresses or rejecting invalid\n"
        "ones. Other string rules still validate fine."),
    contract=contract("email format rule", """Symptom: validation aborts with a `panic:` instead of returning an error or accepting the struct.

A bound string field annotated `binding:"email"` must be accepted exactly when
the value is a valid RFC-5322-style address as interpreted by the bound
validation engine:

- Shape is `local@domain` with exactly one `@`. The domain is dot-separated
  labels and MUST contain at least one dot — a single-label domain (`a@b`)
  is rejected, as is a trailing-dot domain (`a@b.co.`).
- The FINAL label (TLD) must begin AND end with a letter (Unicode letters
  count) and may contain letters, digits and hyphens internally —
  `a@b.c3x` is valid; `a@b.c-3`, `a@b.23`, `a@b.2c`, `a@b.cx-` are not.
  Mid-domain labels may begin with digits (`a@1b.cd` valid) but empty
  mid-labels are rejected (`a@sub..b.co`).
- Local part: dot-atom (`a.b.c@d.co` valid; `a..b`, `.a`, `a.` rejected) or a
  QUOTED STRING that must span the whole local part (`"a..b"@c.d`,
  `"a b"@x.co`, `"a@b"@x.co` valid; `"a@b"c@d.co` rejected). Unquoted local
  parts reject spaces and `(),:;`-style specials. Display-name
  (`Name <a@b.co>`) and address-literal (`a@[127.0.0.1]`) forms are rejected.
- Unicode local parts and Unicode domain labels are accepted (`é@b.co`,
  `a@bé.co`, `a@b.cöm`).
- The empty string is rejected.
- Errors identify the field and tag: `failed on the 'email' tag`.""",
        [
            ("TestXRMailfmtBasic", "accepts valid forms incl. unicode/+tags; rejects malformed, single-label domain, TLD-rule violations"),
            ("TestXRMailfmtLocalDots", "local dot-atom: interior dots ok; `..`, leading/trailing dot rejected"),
            ("TestXRMailfmtQuoted", "quoted-string local parts accepted when they span the whole local part"),
            ("TestXRMailfmtProp", "seeded corpus: every accepted address satisfies the structural invariants"),
        ]),
    details=[
        "`email` accepts `a1@b2.com`, `a@b.co`, `a@b-c.d`, `a@1b.cd`, `a@b.c3x`.",
        "`email` rejects `a1@b2.c3` (TLD ends in digit), `a@b.23` (all-digit TLD), `a@b.c-3`, `a@b.2c` (TLD starts digit), `a@b.co.` (trailing dot), `a@b` (single-label domain).",
        "Local dot-atom: `a.b.c@d.co` valid; `a..b`, `.a`, `a.` local parts rejected.",
        "Quoted local parts must span the whole local part: `\"a..b\"@c.d` valid, `\"a@b\"c@d.co` rejected.",
        "Unicode local parts and Unicode domain labels are accepted.",
        "Display-name and domain-literal forms are rejected.",
        "Errors name the field and the `email` tag.",
    ],
    difficulty="""One excised function (`isEmail`) but the contract is a non-obvious
regex boundary: the TLD must begin AND end with a letter while mid-labels may
start with digits — intuition says `a1@b2.c3` is valid (it is not) and
`a@b.co.` is valid (trailing dot — rejected). Quoted local parts, Unicode,
no display-names. The solver must locate the nested-module dep and reproduce
the exact boundary; a plausible-but-wrong email regex fails the seeded corpus.""",
    closure=("`isEmail` (1 func, `baked_in.go`).",
             "`isURLEncoded`/`isHTML`/`regexes.go` emailRegex untouched; tag registry intact — only the email predicate body excised.",
             "`binding.Validator.ValidateStruct` on `binding:\"email\"` string fields; also `Context.ShouldBindJSON`."),
    cheat=gin_cheat("mailfmt", """	rv := reflect.Indirect(reflect.ValueOf(obj))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	var fails []string
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		for _, rule := range strings.Split(f.Tag.Get("binding"), ",") {
			if rule == "email" {
				s := rv.Field(i).String()
				// shallow check: must contain @ and a dot after it
				at := strings.LastIndex(s, "@")
				if at <= 0 || !strings.Contains(s[at:], ".") {
					fails = append(fails, "Key: '"+rt.Name()+"."+f.Name+"' Error:Field validation for '"+f.Name+"' failed on the 'email' tag")
				}
			}
		}
	}
	if len(fails) > 0 {
		return errors.New(strings.Join(fails, "\\n"))
	}
	return nil""",
        note="Deliberately shallow: '@' + a dot only — quoted local parts, Unicode, TLD rules all missed."),
)

UNITS["uuidfmt"] = dict(
    tags="uuid…ulid",
    api=api("uuidfmt", "uuid,uuid3,uuid4,uuid5,uuid_rfc4122,uuid3_rfc4122,uuid4_rfc4122,uuid5_rfc4122,ulid"),
    bug=bugreport(
        "Struct binding crashes on UUID-family format rules. When a bound field\n"
        "carries `uuid`, `uuid3`, `uuid4`, `uuid5`, the `*_rfc4122` variants, or\n"
        "`ulid`, validation aborts with a `panic:` instead of accepting valid\n"
        "identifiers or rejecting malformed ones. Other format rules still work."),
    contract=contract("uuid/ulid format rules", """Symptom: validation aborts with a `panic:` instead of returning an error or accepting the struct.

Bound string fields annotated with the identifier-format rules must be
evaluated exactly:

- `uuid` / `uuid_rfc4122` — the generic 8-4-4-4-12 hyphenated-hex shape; any
  letter case is accepted. Braces, missing hyphens, wrong group lengths and
  non-hex characters are rejected.
- `uuid3` / `uuid5` — lowercase-only hex; the version nibble must be 3 (resp.
  5); the variant nibble must be in `8`,`9`,`a`,`b`. Uppercase is REJECTED.
- `uuid4` — lowercase-only; version nibble `4`; variant nibble `8`/`9`/`a`/`b`.
- `uuid3_rfc4122` — version `3`, ANY case; no variant constraint.
- `uuid4_rfc4122` / `uuid5_rfc4122` — version `4`/`5`, any case; variant nibble
  `8`/`9`/`a`/`b`/`A`/`B`.
- `ulid` — exactly 26 characters from the Crockford alphabet
  `0-9 A-H J-K M-N P-T V-Z` (case-insensitive); `I`, `L`, `O`, `U` are excluded;
  `W` IS included. 25/27 chars or any excluded letter rejects.
- Errors identify the field and tag (`failed on the 'uuid4' tag`).""",
        [
            ("TestXRUuidGeneric", "uuid accepts any-case 8-4-4-4-12 hex; rejects shape violations"),
            ("TestXRUuidVersions", "uuid3/4/5 pin version nibble; uuid4/uuid5 pin variant; lowercase-only"),
            ("TestXRUuidRFC4122", "rfc4122 variants are case-insensitive; v4/v5 keep the variant pin"),
            ("TestXRULID", "ulid accepts 26-char Crockford alphabet incl. W; rejects I/L/O/U and bad length"),
            ("TestXRUuidProp", "seeded hex/hyphen corpus matches the structural oracle"),
        ]),
    details=[
        "`uuid` accepts any-case 8-4-4-4-12 hex; rejects wrong lengths/separators/braces.",
        "`uuid3` requires lowercase + version nibble 3, no variant constraint.",
        "`uuid4`/`uuid5` require lowercase + version nibble + variant in 89ab.",
        "`*_rfc4122` variants allow any case; only v4/v5 rfc4122 pin the variant.",
        "`ulid` accepts exactly 26 chars from `0-9A-HJKMNP-TV-Z` (case-insensitive); rejects I,L,O,U and any other length.",
        "Errors name the field and the specific tag.",
    ],
    difficulty="""Nine excised predicate funcs, one file, nine tag spellings with
near-identical names but THREE different rules: generic (any case), strict
(lowercase + version + variant), rfc4122 (any case, variant only for 4/5).
The case-sensitivity split between `uuid4` and `uuid4_rfc4122` is the classic
trap. `ulid`'s alphabet includes W but excludes I/L/O/U. The seeded sweep
catches any solver that treats the family uniformly.""",
    closure=("`isUUID`, `isUUID3`, `isUUID4`, `isUUID5`, `isUUIDRFC4122`, `isUUID3RFC4122`, `isUUID4RFC4122`, `isUUID5RFC4122`, `isULID` (9 funcs, `baked_in.go`).",
             "regex constants in `regexes.go` intact (uuid/ulid regexes are still referenced by the tag map); the predicate bodies are the closure.",
             "`binding.Validator.ValidateStruct` on string fields with the nine tags."),
    cheat=gin_cheat("uuidfmt", """	rv := reflect.Indirect(reflect.ValueOf(obj))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	var fails []string
	uuidish := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	ulidish := regexp.MustCompile(`^[0-9A-Z]{26}$`)
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		for _, rule := range strings.Split(f.Tag.Get("binding"), ",") {
			s := rv.Field(i).String()
			bad := false
			switch rule {
			case "uuid", "uuid3", "uuid4", "uuid5", "uuid_rfc4122", "uuid3_rfc4122", "uuid4_rfc4122", "uuid5_rfc4122":
				bad = !uuidish.MatchString(s) // version/variant/case rules all missed
			case "ulid":
				bad = !ulidish.MatchString(s) // alphabet misses I/L/O/U exclusion
			}
			if bad {
				fails = append(fails, "Key: '"+rt.Name()+"."+f.Name+"' Error:Field validation for '"+f.Name+"' failed on the '"+rule+"' tag")
			}
		}
	}
	if len(fails) > 0 {
		return errors.New(strings.Join(fails, "\\n"))
	}
	return nil""",
        note="Deliberately shallow: one generic uuid regex for all 8 tags, permissive ULID — version/variant/case rules all missed.", add_imports=("errors", "regexp")),
)

UNITS["urifmt"] = dict(
    api=api("urifmt", "uri,url,http_url,https_url,urn_rfc2141,datauri"),
    bug=bugreport(
        "Struct binding crashes on URI/URL-family format rules. When a bound\n"
        "field carries `uri`, `url`, `http_url`, `https_url`, `urn_rfc2141` or\n"
        "`datauri`, validation aborts with a `panic:` instead of accepting\n"
        "valid references or rejecting malformed ones."),
    contract=contract("uri/url/urn/datauri format rules", """Symptom: validation aborts with a `panic:` instead of returning an error or accepting the struct.

Bound string fields annotated with the reference-format rules must be
evaluated exactly:

- `uri` — a valid URI *reference* per RFC 3986 request-URI parsing: absolute
  paths like `/abs/path` and full URLs are accepted; a bare fragment
  (`#only-frag`) is rejected (the reference is empty after the fragment is
  removed); spaces and empty strings rejected.
- `url` — a parsed URL that is absolute: a scheme is required; `mailto:a@b.co`
  (opaque) and `file:///etc/passwd` pass; `file://` (no host, no opaque part)
  and bare paths fail.
- `http_url` — scheme `http` OR `https` (case-insensitive) AND a non-empty
  host: `https://example.com` passes `http_url`; `http:///path`, `mailto:…`,
  `ftp://…` fail.
- `https_url` — same but `https` only.
- `urn_rfc2141` — `urn:<NID>:<NSS>`; the `urn:` scheme prefix is
  case-insensitive; an empty NID or missing NSS fails.
- `datauri` — `data:[<mediatype>][;base64],<base64-payload>`; the mediatype is
  optional and the `;base64` marker is NOT required — only the payload must be
  valid base64.
- Errors identify the field and tag.""",
        [
            ("TestXRURI", "uri accepts absolute paths and full URIs; rejects bare fragments/empties"),
            ("TestXRURL", "url requires a scheme; opaque and file URLs pass"),
            ("TestXRHTTPURL", "http_url allows http+https with non-empty host; https_url only https"),
            ("TestXRURN", "urn_rfc2141 requires urn:NID:NSS, case-insensitive scheme"),
            ("TestXRDataURI", "datauri requires base64 payload; mediatype and ;base64 marker optional"),
            ("TestXRURLProp", "seeded scheme+tail corpus matches the host-nonempty oracle"),
        ]),
    details=[
        "`uri` accepts `/abs/path`, `http://x/y`, `mailto:`, `urn:`; rejects `#frag`-only, spaces, empty.",
        "`url` requires a scheme; `file://` fails, `file:///p` passes.",
        "`http_url` accepts http AND https with a non-empty host; `https_url` accepts only https.",
        "`urn_rfc2141` needs `urn:<nid>:<nss>`; scheme case-insensitive.",
        "`datauri` requires a base64 payload after the comma; mediatype/`;base64` optional.",
        "Errors name the field and tag.",
    ],
    difficulty="""Six excised predicates across two parse strategies
(ParseRequestURI vs Parse vs regex). The traps are subtle: `uri` accepts
absolute paths but rejects fragment-only refs; `http_url` accepts `https`
(scheme-contains check) but demands a host; `datauri` ignores the `;base64`
marker and only checks the payload; `urn:` scheme is case-insensitive while
NID/NSS have their own grammar. A solver guessing 'URL = must have host' fails
`mailto:`/`file:///`.""",
    closure=("`isURI`, `isURL`, `isHttpURL`, `isHttpsURL`, `isUrnRFC2141`, `isDataURI` (6 funcs, `baked_in.go`).",
             "`isURN`/`isURL`-adjacent helpers and all regexes intact.",
             "`binding.Validator.ValidateStruct` on string fields with the six tags."),
    cheat=gin_cheat("urifmt", """	rv := reflect.Indirect(reflect.ValueOf(obj))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	var fails []string
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		for _, rule := range strings.Split(f.Tag.Get("binding"), ",") {
			s := rv.Field(i).String()
			bad := false
			switch rule {
			case "uri", "url":
				bad = !strings.Contains(s, "://") // shallow: opaque/absolute-path missed
			case "http_url":
				bad = !strings.HasPrefix(s, "http://")
			case "https_url":
				bad = !strings.HasPrefix(s, "https://")
			case "urn_rfc2141":
				bad = !strings.HasPrefix(s, "urn:")
			case "datauri":
				bad = !strings.HasPrefix(s, "data:") || !strings.Contains(s, ";base64,")
			}
			if bad {
				fails = append(fails, "Key: '"+rt.Name()+"."+f.Name+"' Error:Field validation for '"+f.Name+"' failed on the '"+rule+"' tag")
			}
		}
	}
	if len(fails) > 0 {
		return errors.New(strings.Join(fails, "\\n"))
	}
	return nil""",
        note="Deliberately shallow: '://' checks, http-only for http_url, requires ;base64 — every contract trap missed."),
)

UNITS["ipcidr"] = dict(
    api=api("ipcidr", "ipv4,ipv6,ip,cidrv4,cidrv6,cidr,mac"),
    bug=bugreport(
        "Struct binding crashes on IP/CIDR/MAC format rules. When a bound field\n"
        "carries `ipv4`, `ipv6`, `ip`, `cidrv4`, `cidrv6`, `cidr` or `mac`,\n"
        "validation aborts with a `panic:` instead of accepting valid addresses\n"
        "or rejecting malformed ones."),
    contract=contract("ip/cidr/mac format rules", """Symptom: validation aborts with a `panic:` instead of returning an error or accepting the struct.

Bound string fields annotated with the network-address rules must be
evaluated exactly:

- `ipv4` — dotted-quad IPv4; a v4-mapped v6 literal (`::ffff:1.2.3.4`) is
  ACCEPTED (it converts to v4); out-of-range octets, missing octets, `::1`
  rejected.
- `ipv6` — a parseable v6 literal that is NOT v4-convertible: `::ffff:1.2.3.4`
  is REJECTED here (the inverse trap).
- `ip` — either family (v4-mapped counts).
- `cidr` — any `net.ParseCIDR`-able value; host bits are allowed
  (`10.0.0.1/8` passes).
- `cidrv4` — a parseable CIDR whose address is v4 AND equals the masked network
  address: `10.0.0.0/8` passes, `10.0.0.1/8` FAILS (host bits set).
- `cidrv6` — a parseable CIDR whose address is not v4-convertible; host bits
  are ALLOWED (`2001:db8::1/32` passes — asymmetric with cidrv4).
- `mac` — anything `net.ParseMAC` accepts: `aa:bb:…`, `aa-bb-…`, `aabb.ccdd.eeff`
  dot form, and the bare 12-hex form `aabbccddeeff` all pass.
- Errors identify the field and tag.""",
        [
            ("TestXRIPv4", "ipv4 accepts dotted-quad and v4-mapped v6; rejects range/shape violations"),
            ("TestXRIPv6", "ipv6 rejects v4-mapped literals — the inverse trap"),
            ("TestXRIPGeneric", "ip accepts both families"),
            ("TestXRCIDR", "cidr accepts host bits set"),
            ("TestXRCIDRv4v6", "cidrv4 requires the masked network address; cidrv6 does not — the asymmetry"),
            ("TestXRMAC", "mac accepts colon/hyphen/dot/bare-12-hex forms"),
            ("TestXRIPv4Prop", "seeded octet corpus matches the dotted-quad oracle"),
        ]),
    details=[
        "`ipv4` accepts dotted quad + v4-mapped v6 (`::ffff:1.2.3.4`); rejects range/shape violations.",
        "`ipv6` REJECTS v4-mapped literals; accepts any non-v4 v6.",
        "`cidr` accepts host bits (`10.0.0.1/8`); `cidrv4` requires the network address.",
        "`cidrv6` allows host bits — asymmetric with cidrv4.",
        "`mac` accepts colon, hyphen, dot and bare 12-hex forms.",
        "Errors name the field and tag.",
    ],
    difficulty="""Seven predicates with THREE different strictness levels that
don't line up with intuition: `ip` vs `ipv6` disagree on v4-mapped literals,
`cidr` vs `cidrv4` disagree on host bits, and `cidrv6` is asymmetric (no
network-address check). ParseMAC accepts a bare-12-hex form nobody expects.
The seeded ipv4 corpus pins the octet grammar (leading zeros, range).""",
    closure=("`isIPv4`, `isIPv6`, `isIP`, `isCIDRv4`, `isCIDRv6`, `isCIDR`, `isMAC` (7 funcs, `baked_in.go`).",
             "network-resolution tags (`ip4_addr`/`ip6_addr`/`ip_addr`/`hostname_port`-adjacent) untouched — those need DNS and are excluded from this unit by design.",
             "`binding.Validator.ValidateStruct` on string fields with the seven tags."),
    cheat=gin_cheat("ipcidr", """	rv := reflect.Indirect(reflect.ValueOf(obj))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	var fails []string
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		for _, rule := range strings.Split(f.Tag.Get("binding"), ",") {
			s := rv.Field(i).String()
			bad := false
			switch rule {
			case "ipv4":
				bad = net.ParseIP(s) == nil || strings.Contains(s, ":")
			case "ipv6":
				bad = !strings.Contains(s, ":")
			case "ip":
				bad = net.ParseIP(s) == nil
			case "cidr", "cidrv4", "cidrv6":
				_, _, e := net.ParseCIDR(s)
				bad = e != nil // misses the v4/v6 family + host-bit asymmetries
			case "mac":
				_, e := net.ParseMAC(s)
				bad = e != nil
			}
			if bad {
				fails = append(fails, "Key: '"+rt.Name()+"."+f.Name+"' Error:Field validation for '"+f.Name+"' failed on the '"+rule+"' tag")
			}
		}
	}
	if len(fails) > 0 {
		return errors.New(strings.Join(fails, "\\n"))
	}
	return nil""",
        note="Deliberately shallow: 'contains colon' v6 check, no family/host-bit asymmetry.", add_imports=("errors", "net")),
)

UNITS["hostport"] = dict(
    api=api("hostport", "hostname,hostname_rfc1123,fqdn,dns_rfc1035_label,hostname_port,port"),
    bug=bugreport(
        "Struct binding crashes on hostname/port format rules. When a bound\n"
        "field carries `hostname`, `hostname_rfc1123`, `fqdn`,\n"
        "`dns_rfc1035_label`, `hostname_port` or `port`, validation aborts with\n"
        "a `panic:` instead of accepting valid names or rejecting malformed\n"
        "ones."),
    contract=contract("hostname/port format rules", """Symptom: validation aborts with a `panic:` instead of returning an error or accepting the struct.

Bound fields annotated with the host/port rules must be evaluated exactly:

- `hostname` (RFC 952) — dot-separated labels, each starting with a LETTER,
  then letters/digits/hyphens, ≤63 chars per label. `1abc` FAILS.
- `hostname_rfc1123` — same but labels may start with a digit (`1abc`,
  `9.9.9.9` pass). Leading/trailing dash, underscores, >63-char labels fail.
- `fqdn` — at least two labels AND the final label must start with a letter;
  a trailing dot is allowed. `example` and `example.123` fail.
- `dns_rfc1035_label` — a single label, lowercase-only, starts with a letter,
  ends alphanumeric, ≤63 chars; `a`, `a-b`, `a1-b2` pass; `ABC`, `-abc`,
  `abc-`, `1abc` fail.
- `hostname_port` — `host:port` where port is numeric 1-65535 and host is
  empty or RFC1123-valid: `:8080` passes (empty host), `example.com:0`,
  `example.com:65536`, non-numeric ports and double colons fail.
- `port` (uint field) — 1 ≤ value ≤ 65535; `0` and `65536` fail.
- Errors identify the field and tag.""",
        [
            ("TestXRHostname", "hostname labels start with a letter; digits rejected"),
            ("TestXRHostname1123", "hostname_rfc1123 allows leading digits"),
            ("TestXRFQDN", "fqdn needs ≥2 labels and an alpha-leading TLD; trailing dot ok"),
            ("TestXRDNSLabel", "dns_rfc1035_label is lowercase-only, letter-leading, ≤63"),
            ("TestXRHostnamePort", "hostname_port allows empty host, port 1-65535"),
            ("TestXRPort", "port uint rule is 1-65535"),
            ("TestXRHostnameProp", "seeded label corpus matches the RFC952 oracle"),
        ]),
    details=[
        "`hostname` requires letter-leading labels (RFC952); `hostname_rfc1123` allows digits.",
        "`fqdn` requires ≥2 labels and a letter-leading final label; trailing dot allowed.",
        "`dns_rfc1035_label` is lowercase-only, letter-leading, ≤63 chars.",
        "`hostname_port` accepts `:8080` (empty host); port must be 1-65535.",
        "`port` on a uint field is 1-65535 inclusive.",
        "Errors name the field and tag.",
    ],
    difficulty="""Six predicates, four different label grammars (RFC952 vs
RFC1123 vs FQDN-with-alpha-TLD vs lowercase-DNS1035) — a solver that treats
'hostname' as one rule fails the leading-digit and TLD splits. `hostname_port`
allows an empty host; `port` is a numeric (not string) rule with an inclusive
65535 cap. The seeded hostname corpus pins label-length and dash edges.""",
    closure=("`isHostnameRFC952`, `isHostnameRFC1123`, `isFQDN`, `isDnsRFC1035LabelFormat`, `isHostnamePort`, `isPort` (6 funcs, `baked_in.go`).",
             "hostname/port regexes in `regexes.go` intact; DNS-resolution tags untouched.",
             "`binding.Validator.ValidateStruct` on string fields (five tags) and a uint field (`port`)."),
    cheat=gin_cheat("hostport", """	rv := reflect.Indirect(reflect.ValueOf(obj))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	var fails []string
	hostish := regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		for _, rule := range strings.Split(f.Tag.Get("binding"), ",") {
			fld := rv.Field(i)
			bad := false
			switch rule {
			case "hostname", "hostname_rfc1123", "fqdn", "dns_rfc1035_label":
				s := fld.String()
				bad = s == "" || !hostish.MatchString(s) || len(s) > 253 // no per-label/TLD/letter rules
			case "hostname_port":
				s := fld.String()
				h, p, ok := strings.Cut(s, ":")
				bad = !ok || p == "" || h == "" || !hostish.MatchString(h) // empty host wrongly rejected
				if !bad {
					if n, e := strconv.Atoi(p); e != nil || n < 1 || n > 65535 {
						bad = true
					}
				}
			case "port":
				v := fld.Uint()
				bad = v < 1 || v > 65535
			}
			if bad {
				fails = append(fails, "Key: '"+rt.Name()+"."+f.Name+"' Error:Field validation for '"+f.Name+"' failed on the '"+rule+"' tag")
			}
		}
	}
	if len(fails) > 0 {
		return errors.New(strings.Join(fails, "\\n"))
	}
	return nil""",
        note="Deliberately shallow: single permissive host regex, empty-host wrongly rejected, no TLD/label rules.", add_imports=("errors", "regexp")),
)

UNITS["contain"] = dict(
    api=api("contain", "contains,containsany,containsrune,fieldcontains"),
    bug=bugreport(
        "Struct binding crashes on substring-containment rules. When a bound\n"
        "field carries `contains`, `containsany`, `containsrune` or\n"
        "`fieldcontains`, validation aborts with a `panic:` instead of\n"
        "checking the value."),
    contract=contract("containment rules", """Symptom: validation aborts with a `panic:` instead of returning an error or accepting the struct.

Bound string fields annotated with the containment rules must be evaluated
exactly:

- `contains=<sub>` — `strings.Contains` semantics; `contains` with an EMPTY
  param always passes (every string contains "").
- `containsany=<chars>` — `strings.ContainsAny`: passes when the field holds
  at least one char from the set; an EMPTY charset always FAILS.
- `containsrune=<s>` — checks only the FIRST rune of the param:
  `containsrune=öx` requires `ö`, `x` alone is not enough.
- `fieldcontains=<FieldName>` — the field's value must contain the sibling's
  current string value; an empty sibling is always contained (passes); a
  sibling name that does not exist FAILS.
- Matching is case-sensitive, works across newlines, and applies to the whole
  string (prefix/suffix/middle all count).
- Errors identify the field and tag.""",
        [
            ("TestXRContains", "contains uses substring semantics, case-sensitive"),
            ("TestXRContainsEmptyParam", "bare contains with empty param always passes"),
            ("TestXRContainsAny", "containsany needs one charset char; empty charset always fails"),
            ("TestXRContainsRune", "containsrune checks only the first rune of the param"),
            ("TestXRFieldContains", "fieldcontains compares against the sibling's value; missing sibling fails"),
            ("TestXRContainsProp", "seeded corpus matches strings.Contains"),
        ]),
    details=[
        "`contains=<sub>` = substring check; empty param always passes.",
        "`containsany=<chars>` = ContainsAny; empty charset always fails.",
        "`containsrune=<s>` checks only the param's first rune.",
        "`fieldcontains=F` requires the sibling's value inside the field; missing sibling fails.",
        "Case-sensitive; whole-string search.",
        "Errors name the field and tag.",
    ],
    difficulty="""Four predicates where the EMPTY PARAM inverts intuition in
opposite directions (`contains` passes, `containsany` fails), `containsrune`
silently checks only the first rune of a multi-rune param, and `fieldcontains`
crosses into sibling lookup (missing sibling → fail, empty sibling → pass).
The seeded corpus is a straight `strings.Contains` oracle — but only after
the solver gets all four edge rules right.""",
    closure=("`contains`, `containsAny`, `containsRune`, `fieldContains` (4 funcs, `baked_in.go`).",
             "`excludes*`/`startsWith`-family and field-lookup plumbing intact.",
             "`binding.Validator.ValidateStruct` on string fields; `fieldcontains` resolves a sibling field name."),
    cheat=gin_cheat("contain", """	rv := reflect.Indirect(reflect.ValueOf(obj))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	var fails []string
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		for _, rule := range strings.Split(f.Tag.Get("binding"), ",") {
			name, param, _ := strings.Cut(rule, "=")
			fld := rv.Field(i)
			bad := false
			switch name {
			case "contains":
				bad = param != "" && !strings.Contains(fld.String(), param)
			case "containsany":
				bad = !strings.ContainsAny(fld.String(), param)
			case "containsrune":
				bad = !strings.ContainsRune(fld.String(), rune(param[0])) // byte, not rune
			case "fieldcontains":
				sib := rv.FieldByName(param)
				bad = !sib.IsValid() || !strings.Contains(fld.String(), sib.String())
			}
			if bad {
				fails = append(fails, "Key: '"+rt.Name()+"."+f.Name+"' Error:Field validation for '"+f.Name+"' failed on the '"+name+"' tag")
			}
		}
	}
	if len(fails) > 0 {
		return errors.New(strings.Join(fails, "\\n"))
	}
	return nil""",
        note="Deliberately shallow: empty-param edge missed, rune taken as byte, empty-sibling edge missed."),
)

UNITS["exclude"] = dict(
    api=api("exclude", "excludes,excludesall,excludesrune,fieldexcludes"),
    bug=bugreport(
        "Struct binding crashes on substring-exclusion rules. When a bound\n"
        "field carries `excludes`, `excludesall`, `excludesrune` or\n"
        "`fieldexcludes`, validation aborts with a `panic:` instead of\n"
        "checking the value."),
    contract=contract("exclusion rules", """Symptom: validation aborts with a `panic:` instead of returning an error or accepting the struct.

Bound string fields annotated with the exclusion rules must be evaluated
exactly — the polarity mirror of the `contains` family:

- `excludes=<sub>` — the field must NOT contain the substring; an EMPTY param
  always FAILS (every string contains "").
- `excludesall=<chars>` — the field must not contain ANY charset char; an
  EMPTY charset always PASSES (ContainsAny("") is false → nothing excluded).
- `excludesrune=<s>` — forbids only the FIRST rune of the param.
- `fieldexcludes=<FieldName>` — the field must not contain the sibling's
  current value; a sibling name that does not exist PASSES (there is nothing
  to exclude — the mirror-image trap of `fieldcontains`); an empty sibling is
  always contained → always FAILS.
- Case-sensitive; empty field passes `excludes`/`excludesall`/`excludesrune`.
- Errors identify the field and tag.""",
        [
            ("TestXRExcludes", "excludes = not-contains; empty field passes"),
            ("TestXRExcludesEmptyParam", "bare excludes with empty param always fails"),
            ("TestXRExcludesAll", "excludesall forbids any charset char; empty charset always passes"),
            ("TestXRExcludesRune", "excludesrune forbids only the param's first rune"),
            ("TestXRFieldExcludes", "fieldexcludes: missing sibling passes, empty sibling fails"),
            ("TestXRExcludesProp", "seeded corpus matches !ContainsAny"),
        ]),
    details=[
        "`excludes=<sub>` = !Contains; EMPTY param always fails.",
        "`excludesall=<chars>` = !ContainsAny; EMPTY charset always passes.",
        "`excludesrune=<s>` forbids only the param's first rune.",
        "`fieldexcludes=F`: missing sibling PASSES; empty sibling always fails.",
        "Case-sensitive; empty field passes the non-sibling rules.",
        "Errors name the field and tag.",
    ],
    difficulty="""The polarity mirror of `contain` — and every empty-param edge
flips: `excludes=` fails always, `excludesall=` passes always. The killer pair
is `fieldcontains` vs `fieldexcludes` on a MISSING sibling: one fails, the
other passes. A solver who mirrors the code but gets one default wrong fails
the seeded corpus.""",
    closure=("`excludes`, `excludesAll`, `excludesRune`, `fieldExcludes` (4 funcs, `baked_in.go`).",
             "`contains*`/`startsNotWith`-family intact.",
             "`binding.Validator.ValidateStruct` on string fields; `fieldexcludes` resolves a sibling name."),
    cheat=gin_cheat("exclude", """	rv := reflect.Indirect(reflect.ValueOf(obj))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	var fails []string
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		for _, rule := range strings.Split(f.Tag.Get("binding"), ",") {
			name, param, _ := strings.Cut(rule, "=")
			fld := rv.Field(i)
			bad := false
			switch name {
			case "excludes":
				bad = strings.Contains(fld.String(), param) // empty param → true → fails always (correct by accident)
			case "excludesall":
				bad = param != "" && strings.ContainsAny(fld.String(), param) // empty → pass (correct by accident)
			case "excludesrune":
				bad = strings.ContainsRune(fld.String(), rune(param[0]))
			case "fieldexcludes":
				sib := rv.FieldByName(param)
				bad = sib.IsValid() && strings.Contains(fld.String(), sib.String())
				// empty sibling: Contains(x,"")=true → bad — but the cheat
				// only reaches here via panic recover; the missing-sibling
				// pass edge is covered, the empty-sibling FAIL edge too.
			}
			if bad {
				fails = append(fails, "Key: '"+rt.Name()+"."+f.Name+"' Error:Field validation for '"+f.Name+"' failed on the '"+name+"' tag")
			}
		}
	}
	if len(fails) > 0 {
		return errors.New(strings.Join(fails, "\\n"))
	}
	return nil""",
        note="Deliberately shallow mirror; misses first-rune-of-param and non-string sibling coercion."),
)

UNITS["affix"] = dict(
    api=api("affix", "startswith,endswith,startsnotwith,endsnotwith"),
    bug=bugreport(
        "Struct binding crashes on prefix/suffix rules. When a bound field\n"
        "carries `startswith`, `endswith`, `startsnotwith` or `endsnotwith`,\n"
        "validation aborts with a `panic:` instead of checking the value."),
    contract=contract("prefix/suffix rules", """Symptom: validation aborts with a `panic:` instead of returning an error or accepting the struct.

Bound string fields annotated with the affix rules must be evaluated exactly:

- `startswith=<p>` / `endswith=<s>` — `strings.HasPrefix`/`HasSuffix`;
  case-sensitive; an EMPTY param always PASSES (every string has the empty
  affix).
- `startsnotwith=<p>` / `endsnotwith=<s>` — the negations; an EMPTY param
  always FAILS (the empty affix is always present).
- An empty field has no non-empty prefix/suffix: it fails `startswith=x` and
  passes `startsnotwith=x`.
- The whole affix must match at the boundary — mid-string occurrences don't
  count.
- Errors identify the field and tag.""",
        [
            ("TestXRStartsEndsWith", "startswith/endswith are boundary checks, case-sensitive"),
            ("TestXRAffixEmptyParams", "empty param passes the positive rules, fails the negations"),
            ("TestXRStartsEndsNotWith", "startsnotwith/endsnotwith invert on boundary match"),
            ("TestXRAffixProp", "seeded corpus matches strings.HasPrefix"),
        ]),
    details=[
        "`startswith`/`endswith` = HasPrefix/HasSuffix, case-sensitive.",
        "Empty param always passes the positive rules.",
        "`startsnotwith`/`endsnotwith` are the negations; empty param always fails.",
        "Empty field fails `startswith=x`, passes `startsnotwith=x`.",
        "Mid-string occurrences don't count.",
        "Errors name the field and tag.",
    ],
    difficulty="""Four tiny predicates whose contract is almost entirely the
empty-param edges: `startswith=` passes where `startsnotwith=` fails, and the
empty FIELD splits the same way. Negations on boundaries are easy to get
backwards under pressure. Small closure, small suite — the trap density is
the point.""",
    closure=("`startsWith`, `endsWith`, `startsNotWith`, `endsNotWith` (4 funcs, `baked_in.go`).",
             "`contains`/`excludes` families intact.",
             "`binding.Validator.ValidateStruct` on string fields."),
    cheat=gin_cheat("affix", """	rv := reflect.Indirect(reflect.ValueOf(obj))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	var fails []string
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		for _, rule := range strings.Split(f.Tag.Get("binding"), ",") {
			name, param, _ := strings.Cut(rule, "=")
			s := rv.Field(i).String()
			bad := false
			switch name {
			case "startswith":
				bad = !strings.HasPrefix(s, param)
			case "endswith":
				bad = !strings.HasSuffix(s, param)
			case "startsnotwith", "endsnotwith":
				// skipped: negation tags unhandled — negation asserts fail
			}
			if bad {
				fails = append(fails, "Key: '"+rt.Name()+"."+f.Name+"' Error:Field validation for '"+f.Name+"' failed on the '"+name+"' tag")
			}
		}
	}
	if len(fails) > 0 {
		return errors.New(strings.Join(fails, "\\n"))
	}
	return nil""",
        note="Deliberately shallow: negation tags skipped entirely — startsnotwith/endsnotwith assertions fail."),
)

UNITS["crossfld"] = dict(
    api=api("crossfld", "eqfield,nefield,ltfield,ltefield,gtfield,gtefield"),
    bug=bugreport(
        "Struct binding crashes on same-struct cross-field comparison rules.\n"
        "When a bound field carries `eqfield`, `nefield`, `ltfield`, `ltefield`,\n"
        "`gtfield` or `gtefield`, validation aborts with a `panic:` instead of\n"
        "comparing the fields."),
    contract=contract("same-struct cross-field comparisons", """Symptom: validation aborts with a `panic:` instead of returning an error or accepting the struct.

Bound fields annotated with the same-struct comparison rules must be
evaluated exactly; the param names a sibling field:

- `eqfield=<F>` / `nefield=<F>` — equality by KIND: ints/uints/floats/bools by
  value, slices/maps/arrays by LENGTH, `time.Time` by instant, strings by
  VALUE (not length).
- `ltfield`/`ltefield`/`gtfield`/`gtefield` — ordering by KIND: numerics by
  value, `time.Time` by before/after, and — the trap — strings by LENGTH
  (`"z" ltfield "aa"` passes), NOT lexical order.
- A missing sibling or a kind mismatch: `eqfield`/`lt*`/`gt*` FAIL,
  `nefield` PASSES.
- `ltfield`/`gtfield` on slices fall through to the string-repr comparison —
  they do NOT compare `Len()` (unlike `eqfield` on slices).
- Errors identify the field and tag.""",
        [
            ("TestXREqField", "eqfield compares by kind: ints/bools by value, strings by VALUE"),
            ("TestXRNeField", "nefield inverts; missing sibling and kind mismatch pass"),
            ("TestXREqFieldMissing", "eqfield fails on missing sibling or kind mismatch"),
            ("TestXRLtFieldStrings", "ltfield on strings compares LENGTH, not lexical order"),
            ("TestXRLteGteField", "ltefield/gtefield are inclusive numeric comparisons"),
            ("TestXRCrossFieldSpecialKinds", "slices compare Len for eq; time compares instants"),
            ("TestXREqFieldProp", "seeded int corpus matches =="),
        ]),
    details=[
        "`eqfield`/`nefield` compare by kind: numeric/bool value, string VALUE, slice/map/array length, time instant.",
        "`ltfield`/`gtfield` on strings compare LENGTH (not lexical).",
        "`lt*`/`gt*` on slices do NOT compare Len — they hit the string-repr default.",
        "Missing sibling or kind mismatch: eq/lt/lte/gt/gte fail; ne passes.",
        "time.Time compares instants.",
        "Errors name the field and tag.",
    ],
    difficulty="""Six predicates whose kind-switch is the contract: the same tag
means value-compare on ints, LENGTH-compare on strings, instant-compare on
time.Time, and Len-compare for eq/ne on slices — but NOT for lt/gt (those hit
the string-repr default). Missing-sibling defaults differ per polarity. A
solver who writes a lexical string compare passes most cases and fails the
`"z" ltfield "aa"` trap.""",
    closure=("`isEqField`, `isNeField`, `isLtField`, `isLteField`, `isGtField`, `isGteField` (6 funcs, `baked_in.go`).",
             "`GetStructFieldOK`/`getStructFieldOKInternal` resolution plumbing intact; the `*csfield` family is a separate unit.",
             "`binding.Validator.ValidateStruct` on int/string/bool/slice/time.Time fields with sibling params."),
    cheat=gin_cheat("crossfld", """	rv := reflect.Indirect(reflect.ValueOf(obj))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	var fails []string
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		for _, rule := range strings.Split(f.Tag.Get("binding"), ",") {
			name, param, _ := strings.Cut(rule, "=")
			sib := rv.FieldByName(param)
			fld := rv.Field(i)
			bad := false
			switch name {
			case "eqfield":
				bad = !sib.IsValid() || fmt.Sprint(fld.Interface()) != fmt.Sprint(sib.Interface())
			case "nefield":
				bad = sib.IsValid() && fmt.Sprint(fld.Interface()) == fmt.Sprint(sib.Interface())
			case "ltfield":
				bad = !sib.IsValid() || !(fmt.Sprint(fld.Interface()) < fmt.Sprint(sib.Interface())) // lexical, not len
			case "ltefield":
				bad = !sib.IsValid() || !(fmt.Sprint(fld.Interface()) <= fmt.Sprint(sib.Interface()))
			case "gtfield":
				bad = !sib.IsValid() || !(fmt.Sprint(fld.Interface()) > fmt.Sprint(sib.Interface()))
			case "gtefield":
				bad = !sib.IsValid() || !(fmt.Sprint(fld.Interface()) >= fmt.Sprint(sib.Interface()))
			}
			if bad {
				fails = append(fails, "Key: '"+rt.Name()+"."+f.Name+"' Error:Field validation for '"+f.Name+"' failed on the '"+name+"' tag")
			}
		}
	}
	if len(fails) > 0 {
		return errors.New(strings.Join(fails, "\\n"))
	}
	return nil""",
        note="Deliberately shallow: fmt.Sprint lexical compare — string-len ordering, slice-Len, time instants and numeric value all missed.", add_imports=("errors", "fmt")),
)

UNITS["xstruct"] = dict(
    api=api("xstruct", "eqcsfield,necsfield,ltcsfield,ltecsfield,gtcsfield,gtecsfield"),
    bug=bugreport(
        "Struct binding crashes on cross-struct field comparison rules. When a\n"
        "bound field carries `eqcsfield`, `necsfield`, `ltcsfield`,\n"
        "`ltecsfield`, `gtcsfield` or `gtecsfield`, validation aborts with a\n"
        "`panic:` instead of comparing the fields."),
    contract=contract("cross-struct field comparisons", """Symptom: validation aborts with a `panic:` instead of returning an error or accepting the struct.

Bound fields annotated with the cross-struct comparison rules must be
evaluated exactly; the param is a DOTTED PATH resolved against the struct
that directly contains the field:

- `eqcsfield=<A.B.C>` walks the path (`Inner.X`, `Inner.Deep.X`); each step
  dereferences pointers — a nil intermediate pointer makes resolution FAIL.
- `eqcsfield`/`necsfield` compare by kind, like `eqfield`: strings by VALUE.
- `ltcsfield`/`ltecsfield`/`gtcsfield`/`gtecsfield` — numerics by value,
  `time.Time` by instant, and — the second trap, opposite of `ltfield` —
  strings by LEXICAL order (`"ab" ltcsfield "za"` passes).
- A missing path element, a nil pointer hop, or a kind mismatch:
  `eqcsfield`/`lt*`/`gt*` FAIL, `necsfield` PASSES.
- Errors identify the field and tag.""",
        [
            ("TestXREqCsField", "eqcsfield resolves dotted paths; compares by kind"),
            ("TestXRCsPathMissing", "missing path element fails eq, passes ne"),
            ("TestXRCsPtrTraversal", "nil pointer hops fail resolution"),
            ("TestXRCsOrdering", "ltcsfield/gtecsfield numerics; strings are LEXICAL — unlike ltfield"),
            ("TestXRCsDeepPath", "two-level paths resolve"),
            ("TestXRCsTime", "time.Time compares instants"),
            ("TestXRCsProp", "seeded int corpus matches == and <"),
        ]),
    details=[
        "`*csfield` params are dotted paths resolved against the containing struct.",
        "Each hop dereferences pointers; a nil hop fails resolution.",
        "Missing path or kind mismatch: eq/lt/lte/gt/gte fail, ne passes.",
        "String ordering for `*csfield` is LEXICAL — opposite of `*field`'s length.",
        "time.Time compares instants.",
        "Errors name the field and tag.",
    ],
    difficulty="""Same six-predicate kind-switch as `crossfld` but the param is a
dotted path (pointer dereference per hop, nil-hop failure) AND the string
ordering flips to LEXICAL — a solver who reuses the `ltfield` intuition
(len-compare) fails `"ab" ltcsfield "za"`. The two families live in the same
file; keeping their semantics distinct is the trap.""",
    closure=("`isEqCrossStructField`, `isNeCrossStructField`, `isLtCrossStructField`, `isLteCrossStructField`, `isGtCrossStructField`, `isGteCrossStructField` (6 funcs, `baked_in.go`).",
             "same-struct `*field` family and path-resolution plumbing intact.",
             "`binding.Validator.ValidateStruct`; paths resolve within the containing struct via dotted params."),
    cheat=gin_cheat("xstruct", """	rv := reflect.Indirect(reflect.ValueOf(obj))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	var fails []string
	resolve := func(path string) (reflect.Value, bool) {
		cur := rv
		for _, p := range strings.Split(path, ".") {
			if cur.Kind() == reflect.Ptr {
				if cur.IsNil() {
					return reflect.Value{}, false
				}
				cur = cur.Elem()
			}
			if cur.Kind() != reflect.Struct {
				return reflect.Value{}, false
			}
			cur = cur.FieldByName(p)
			if !cur.IsValid() {
				return reflect.Value{}, false
			}
		}
		return cur, true
	}
	_ = resolve
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		for _, rule := range strings.Split(f.Tag.Get("binding"), ",") {
			name, param, _ := strings.Cut(rule, "=")
			sib, ok := resolve(param)
			fld := rv.Field(i)
			bad := false
			switch name {
			case "eqcsfield":
				bad = !ok || fmt.Sprint(fld.Interface()) != fmt.Sprint(sib.Interface())
			case "necsfield":
				bad = ok && fmt.Sprint(fld.Interface()) == fmt.Sprint(sib.Interface())
			case "ltcsfield", "ltecsfield", "gtcsfield", "gtecsfield":
				if !ok {
					bad = true
					break
				}
				a, b := fld.String(), sib.String() // len-compare like *field — WRONG for csfield
				switch name {
				case "ltcsfield":
					bad = !(len(a) < len(b))
				case "ltecsfield":
					bad = !(len(a) <= len(b))
				case "gtcsfield":
					bad = !(len(a) > len(b))
				case "gtecsfield":
					bad = !(len(a) >= len(b))
				}
			}
			if bad {
				fails = append(fails, "Key: '"+rt.Name()+"."+f.Name+"' Error:Field validation for '"+f.Name+"' failed on the '"+name+"' tag")
			}
		}
	}
	if len(fails) > 0 {
		return errors.New(strings.Join(fails, "\\n"))
	}
	return nil""",
        note="Deliberately wrong: reuses the *field* length-compare for strings — csfield is lexical. fmt.Sprint eq misses kind-specific compares.", add_imports=("errors", "fmt")),
)

UNITS["hashfmt"] = dict(
    api=api("hashfmt", "md4,md5,sha256,sha384,sha512,ripemd128,ripemd160,tiger128,tiger160,tiger192"),
    bug=bugreport(
        "Struct binding crashes on cryptographic-hash format rules. When a\n"
        "bound field carries `md4`, `md5`, `sha256`, `sha384`, `sha512`,\n"
        "`ripemd128`, `ripemd160`, `tiger128`, `tiger160` or `tiger192`,\n"
        "validation aborts with a `panic:` instead of checking the hex\n"
        "digest."),
    contract=contract("hash-digest format rules", """Symptom: validation aborts with a `panic:` instead of returning an error or accepting the struct.

Bound string fields annotated with the hash-format rules must be evaluated
exactly. Each rule accepts a lowercase-hex string of EXACTLY one length:

| tag | length |
|---|---|
| `md4`, `md5`, `ripemd128`, `tiger128` | 32 |
| `ripemd160`, `tiger160` | 40 |
| `tiger192` | 48 |
| `sha256` | 64 |
| `sha384` | 96 |
| `sha512` | 128 |

- Hex is LOWERCASE-only (`a-f0-9`): an exact-length UPPERCASE string FAILS
  every rule — the asymmetric trap.
- Off-by-one lengths fail; non-hex chars fail; empty fails.
- Same-length tags are distinct: a correct `md5` implementation does not fix
  `md4`.
- Errors identify the field and tag.""",
        [
            ("TestXRHashLengthMatrix", "each tag accepts exactly its hex length; ±1 and non-hex fail"),
            ("TestXRHashSameLengthDistinctTags", "same-length tags are independent — no shared shortcut"),
            ("TestXRHashProp", "seeded corpus matches the exact-length lowercase-hex oracle"),
        ]),
    details=[
        "`md4`/`md5`/`ripemd128`/`tiger128` = 32 lowercase-hex.",
        "`ripemd160`/`tiger160` = 40; `tiger192` = 48.",
        "`sha256` = 64, `sha384` = 96, `sha512` = 128.",
        "UPPERCASE hex rejected by all ten tags.",
        "Exact-length required; non-hex rejected.",
        "Errors name the field and tag.",
    ],
    difficulty="""Ten predicates that look like one parameterized rule but are
ten separate functions — and the regexes are lowercase-only, so `A-F` fails.
A solver who 'fixes' by accepting any-case hex, or who gets the ripemd/tiger
lengths (40/40/48) wrong, fails the matrix. The same-length quadruple at 32
chars punishes a shared-regex shortcut.""",
    closure=("`isMD4`, `isMD5`, `isSHA256`, `isSHA384`, `isSHA512`, `isRIPEMD128`, `isRIPEMD160`, `isTIGER128`, `isTIGER160`, `isTIGER192` (10 funcs, `baked_in.go`).",
             "all regexes intact — the closure is the predicate bodies only.",
             "`binding.Validator.ValidateStruct` on string fields with the ten tags."),
    cheat=gin_cheat("hashfmt", """	rv := reflect.Indirect(reflect.ValueOf(obj))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	var fails []string
	hexish := regexp.MustCompile(`^[0-9a-fA-F]+$`) // case-insensitive — WRONG
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		for _, rule := range strings.Split(f.Tag.Get("binding"), ",") {
			want := map[string]int{"md4": 32, "md5": 32, "sha256": 64, "sha384": 96, "sha512": 128, "ripemd128": 32, "ripemd160": 40, "tiger128": 32, "tiger160": 40, "tiger192": 48}[rule]
			if want == 0 {
				continue
			}
			s := rv.Field(i).String()
			if len(s) != want || !hexish.MatchString(s) {
				fails = append(fails, "Key: '"+rt.Name()+"."+f.Name+"' Error:Field validation for '"+f.Name+"' failed on the '"+rule+"' tag")
			}
		}
	}
	if len(fails) > 0 {
		return errors.New(strings.Join(fails, "\\n"))
	}
	return nil""",
        note="Deliberately wrong: case-insensitive hex — uppercase should fail.", add_imports=("errors", "regexp")),
)

UNITS["colorfmt"] = dict(
    api=api("colorfmt", "hexcolor,rgb,rgba,hsl,hsla,cmyk"),
    bug=bugreport(
        "Struct binding crashes on CSS-color format rules. When a bound field\n"
        "carries `hexcolor`, `rgb`, `rgba`, `hsl`, `hsla` or `cmyk`,\n"
        "validation aborts with a `panic:` instead of checking the color\n"
        "string."),
    contract=contract("CSS color format rules", """Symptom: validation aborts with a `panic:` instead of returning an error or accepting the struct.

Bound string fields annotated with the color rules must be evaluated exactly:

- `hexcolor` — `#` + 3, 4, 6 or 8 hex digits, any case.
- `rgb` — `rgb(N,N,N)` or `rgb(N%,N%,N%)` with optional inner spaces. Each
  channel is `0|1-99|100-199|200-249|250-255` — so `101%` and even `255%` are
  accepted (the percent cap is 255, not 100); leading zeros (`07`) and mixing
  `%`/plain channels are rejected.
- `rgba` — same channels + alpha `0`, `1` or `0.<digits>`; the `0.` form
  accepts an empty fraction (`rgba(0,0,0,0.)` is VALID); `.5` and `1.5` fail.
- `hsl`/`hsla` — hue `0-360` (no negative, no `361`), saturation/lightness
  `0-100` with MANDATORY `%`; alpha as in `rgba`.
- `cmyk` — `cmyk(N%,N%,N%,N%)`, `N` in `0-100`, `%` mandatory, NO space after
  `(`; `cmyk(0,0,0,0)` fails.
- Errors identify the field and tag.""",
        [
            ("TestXRHexColor", "hexcolor = # + 3/4/6/8 hex digits, any case"),
            ("TestXRRGB", "rgb channels cap at 255 — including the 255% quirk; no mixed forms"),
            ("TestXRRGBA", "rgba alpha is 0|1|0.digits; trailing-dot alpha is valid"),
            ("TestXRHSL", "hsl hue 0-360, s/l need %"),
            ("TestXRHSLA", "hsla adds the rgba-style alpha"),
            ("TestXRCMYK", "cmyk needs four N% components, no space after paren"),
        ]),
    details=[
        "`hexcolor` = `#` + 3/4/6/8 hex, any case.",
        "`rgb` channels 0-255; the `%` form also caps at 255% — `101%` valid.",
        "Leading zeros (`07`) and mixed `%`/plain channels rejected in `rgb`/`rgba`.",
        "`rgba`/`hsla` alpha = `0`, `1`, or `0.<digits>` — `0.` alone valid, `.5` invalid.",
        "`hsl` hue 0-360; `%` mandatory on s/l; `cmyk` needs four `N%`, no space after `(`.",
        "Errors name the field and tag.",
    ],
    difficulty="""Six regexes with four different channel grammars and three
genuine quirks nobody guesses: `rgb(101%,…)` is legal (cap 255 not 100),
`rgba(0,0,0,0.)` is legal (empty fraction), and `cmyk(` forbids the leading
space the other functions allow. The solver must reproduce each regex
boundary exactly — the suite's vectors sit on every quirk.""",
    closure=("`isHEXColor`, `isRGB`, `isRGBA`, `isHSL`, `isHSLA`, `isCMYK` (6 funcs, `baked_in.go`).",
             "color regexes intact; the closure is the six predicates.",
             "`binding.Validator.ValidateStruct` on string fields."),
    cheat=gin_cheat("colorfmt", """	rv := reflect.Indirect(reflect.ValueOf(obj))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	var fails []string
	hexcol := regexp.MustCompile(`^#[0-9a-fA-F]+$`)           // any length — wrong
	rgbish := regexp.MustCompile(`^rgb\\(\\s*\\d+%?,\\s*\\d+%?,\\s*\\d+%?\\s*\\)$`)
	rgbaish := regexp.MustCompile(`^rgba\\(\\s*\\d+%?,\\s*\\d+%?,\\s*\\d+%?,\\s*[0-9.]+\\s*\\)$`)
	hslish := regexp.MustCompile(`^hsl\\(\\s*\\d+,\\s*\\d+%,\\s*\\d+%\\s*\\)$`)
	hslaish := regexp.MustCompile(`^hsla\\(\\s*\\d+,\\s*\\d+%,\\s*\\d+%,\\s*[0-9.]+\\s*\\)$`)
	cmykish := regexp.MustCompile(`^cmyk\\(\\s*\\d+%?,\\s*\\d+%?,\\s*\\d+%?,\\s*\\d+%?\\s*\\)$`)
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		for _, rule := range strings.Split(f.Tag.Get("binding"), ",") {
			s := rv.Field(i).String()
			bad := false
			switch rule {
			case "hexcolor":
				bad = !hexcol.MatchString(s)
			case "rgb":
				bad = !rgbish.MatchString(s)
			case "rgba":
				bad = !rgbaish.MatchString(s)
			case "hsl":
				bad = !hslish.MatchString(s)
			case "hsla":
				bad = !hslaish.MatchString(s)
			case "cmyk":
				bad = !cmykish.MatchString(s)
			}
			if bad {
				fails = append(fails, "Key: '"+rt.Name()+"."+f.Name+"' Error:Field validation for '"+f.Name+"' failed on the '"+rule+"' tag")
			}
		}
	}
	if len(fails) > 0 {
		return errors.New(strings.Join(fails, "\\n"))
	}
	return nil""",
        note="Deliberately shallow: unbounded channel digits, any-length hex, optional % — every regex quirk missed.", add_imports=("errors", "regexp")),
)

UNITS["cryptocoin"] = dict(
    api=api("cryptocoin", "eth_addr,eth_addr_checksum,btc_addr,btc_addr_bech32"),
    bug=bugreport(
        "Struct binding crashes on cryptocurrency-address rules. When a bound\n"
        "field carries `eth_addr`, `eth_addr_checksum`, `btc_addr` or\n"
        "`btc_addr_bech32`, validation aborts with a `panic:` instead of\n"
        "checking the address."),
    contract=contract("cryptocurrency address rules", """Symptom: validation aborts with a `panic:` instead of returning an error or accepting the struct.

Bound string fields annotated with the cryptocurrency rules must be
evaluated exactly:

- `eth_addr` — `0x` + exactly 40 hex chars, any case.
- `eth_addr_checksum` — `eth_addr` shape AND a valid EIP-55 checksum: the
  keccak-256 hash of the lowercase address decides which alpha chars must be
  uppercase. All-lowercase or all-uppercase addresses almost always FAIL; a
  single case flip fails.
- `btc_addr` — base58 P2PKH/P2SH with a real double-SHA256 checksum: known-good
  vectors pass (`1BoatSLRHtKNngkdXEeobR76b53LETtpyT`,
  `3J98t1WpEZ73CNmQviecrnyiWrnqRhWNLy`, `1111111111111111111114oLvT2`); any
  single-char mutation fails; chars outside the base58 alphabet (`0`, `O`,
  `I`, `l`) fail; non-`1`/`3` prefixes fail; bech32 strings fail.
- `btc_addr_bech32` — segwit `bc1…` (or `BC1…`) with a real bech32 polymod:
  BIP-173 vectors pass including witness versions 0-16 and mixed-witness
  lengths; bech32M-encoded vectors FAIL (this is original bech32); wrong-HRP,
  bad-checksum, too-short and base58 strings all fail.
- Errors identify the field and tag.""",
        [
            ("TestXREthAddr", "eth_addr = 0x + 40 hex, any case"),
            ("TestXREthChecksum", "eth_addr_checksum enforces real EIP-55 case positions"),
            ("TestXRBtcAddr", "btc_addr enforces base58 alphabet + double-SHA256 checksum"),
            ("TestXRBtcBech32", "btc_addr_bech32 enforces the original bech32 polymod — bech32m fails"),
        ]),
    details=[
        "`eth_addr` = `0x` + 40 hex, any case.",
        "`eth_addr_checksum` = EIP-55: canonical vectors pass, one case flip fails.",
        "`btc_addr` = base58 P2PKH/P2SH + double-SHA256 checksum; canonical vectors pass, mutations fail.",
        "`btc_addr_bech32` = original bech32 polymod; BIP-173 vectors pass, bech32m fails.",
        "Wrong prefixes/alphabets/lengths rejected.",
        "Errors name the field and tag.",
    ],
    difficulty="""The hardest unit in the family: three real checksums (EIP-55
keccak, base58+double-SHA256, bech32 polymod with witness-version and
length rules). A regex-only 'fix' passes the shape cases and fails every
checksum vector. The bech32-vs-bech32m edge (`BC1SW50QA3JX3S` valid,
`BC1SW50QGDZ25J` invalid) punishes anyone who vendors the wrong variant.
This is the unit where 'the boundary adds discovery' is literal — the solver
must find sha3/base58 plumbing inside the dep module.""",
    closure=("`isEthereumAddress`, `isEthereumAddressChecksum`, `isBitcoinAddress`, `isBitcoinBech32Address` (4 funcs, `baked_in.go`).",
             "sha3/sha256/ripemd160 plumbing in `util.go`, the address regexes, and the bech32 alphabet intact — only the four predicate bodies are excised.",
             "`binding.Validator.ValidateStruct` on string fields with the four tags."),
    cheat=gin_cheat("cryptocoin", """	rv := reflect.Indirect(reflect.ValueOf(obj))
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	var fails []string
	eth := regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	b58 := regexp.MustCompile(`^[13][1-9A-HJ-NP-Za-km-z]{25,34}$`)
	b32 := regexp.MustCompile(`^(?i)bc1[02-9ac-hj-np-z]{7,87}$`)
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		for _, rule := range strings.Split(f.Tag.Get("binding"), ",") {
			s := rv.Field(i).String()
			bad := false
			switch rule {
			case "eth_addr", "eth_addr_checksum":
				bad = !eth.MatchString(s) // no EIP-55 — checksum cases fail
			case "btc_addr":
				bad = !b58.MatchString(s) // no double-sha256 — mutations pass
			case "btc_addr_bech32":
				bad = !b32.MatchString(s) // no polymod — bech32m passes
			}
			if bad {
				fails = append(fails, "Key: '"+rt.Name()+"."+f.Name+"' Error:Field validation for '"+f.Name+"' failed on the '"+rule+"' tag")
			}
		}
	}
	if len(fails) > 0 {
		return errors.New(strings.Join(fails, "\\n"))
	}
	return nil""",
        note="Deliberately shallow: regexes only — all three checksums skipped.", add_imports=("errors", "regexp")),
)

UNITS["structlvl"] = dict(
    api=api("structlvl", "RegisterStructValidation + StructLevel rules"),
    bug=bugreport(
        "Struct-level validation hooks crash. When a caller registers a\n"
        "struct-level validation function through the binding engine's\n"
        "`Engine()` accessor and then validates that struct type, the call\n"
        "aborts with a `panic:` — either at registration or when the struct\n"
        "runs through `binding.Validator.ValidateStruct`. Field-level `binding`\n"
        "tags still work."),
    contract=contract("struct-level validation hooks", """Symptom: registering or running struct-level rules aborts with a `panic:`.

The binding engine exposes its underlying validator via
`binding.Validator.Engine()`. Through it, callers register struct-level rules:

- `RegisterStructValidation(fn, types...)` registers a `func(sl StructLevel)`
  against struct types; it must be usable with plain (non-ctx) functions —
  registration wraps them.
- During `ValidateStruct`, after a struct's field-level tags run, every
  registered struct-level func for that type runs with `sl.Current()` bound to
  the struct under validation — including when the struct is NESTED inside
  another validated value (and `sl.Top()` then reaches the outermost value).
- `sl.ReportError(field, fieldName, structFieldName, tag, param)` must append
  a validation error that names the field and the caller-chosen tag; the error
  surfaces through `ValidateStruct`'s normal error return alongside field-level
  failures.
- Multiple `ReportError` calls accumulate; unregistered types skip silently.""",
        [
            ("TestXRStructLevelBasics", "registered rules run after field tags; ReportError names field+custom tag; errors accumulate"),
            ("TestXRStructLevelNesting", "struct-level funcs fire for nested structs and direct validation"),
        ]),
    details=[
        "`Engine().(*validator.Validate).RegisterStructValidation` registers plain `StructLevelFunc`s.",
        "Struct-level funcs run during `ValidateStruct` after field tags, for registered types.",
        "`sl.ReportError` surfaces errors naming the field and custom tag through the normal error return.",
        "Registered rules also fire when the struct is nested inside another validated value.",
        "Field-level tag errors and struct-level errors accumulate together.",
    ],
    difficulty="""The only unit whose closure spans TWO dep files and an API
surface, not a predicate: `RegisterStructValidation` (+Ctx +MapRules),
`wrapStructLevelFunc`, `ReportError`, `ReportValidationErrors`. The solver
must reconstruct the registration plumbing AND the error-reporting bridge
that appends FieldErrors into the running validation. The panic can fire at
registration time OR at validation time depending on which stub is hit
first — diagnosis alone is a step.""",
    closure=("`wrapStructLevelFunc`, `ReportError`, `ReportValidationErrors` (`struct_level.go`); `RegisterStructValidation`, `RegisterStructValidationCtx`, `RegisterStructValidationMapRules` (`validator_instance.go`).",
             "the `validate` struct machinery, `structLevelFuncs` map, and `validateStruct` execution loop intact — only the registration + reporting surface excised.",
             "`binding.Validator.Engine().(*validator.Validate)` (consumer-sanctioned accessor; in-tree tests use it) + `ValidateStruct`."),
    cheat=gin_cheat("structlvl", """	rv := reflect.Indirect(reflect.ValueOf(obj))
	if rv.Kind() == reflect.Struct && rv.NumField() > 0 {
		return nil // shallow: struct-level rules are treated as satisfied
	}
	return nil""",
        note="Deliberately insufficient: swallows the panic and reports nothing — the excised RegisterStructValidation panics inside the test before any fallback could help.", add_imports=()),
)


def cheat_patch_to_file(text: str) -> str:
    """Normalize the hand-written diff (spaces inside f-string braces)."""
    return text


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--hidden-src", type=Path, default=HIDDEN_SRC)
    ap.add_argument("--families", nargs="*", default=None)
    args = ap.parse_args()
    fams = args.families or sorted(UNITS)
    for fam in fams:
        spec = UNITS[fam]
        a = AUTHOR / fam / "_author"
        (a / "hidden" / "binding").mkdir(parents=True, exist_ok=True)
        (a / "api.md").write_text(spec["api"], encoding="utf-8")
        (a / "bugreport.md").write_text(spec["bug"], encoding="utf-8")
        (a / "contract.md").write_text(spec["contract"], encoding="utf-8")
        (a / "closure.md").write_text(closure(fam, *spec["closure"]), encoding="utf-8")
        (a / "difficulty.md").write_text(difficulty(fam, spec["difficulty"], spec["details"]), encoding="utf-8")
        (a / "DETAILS.md").write_text(details(spec["details"]), encoding="utf-8")
        (a / "cheat.patch").write_text(spec["cheat"], encoding="utf-8")
        hidden = args.hidden_src / f"xr_{fam}_test.go"
        shutil.copy2(hidden, a / "hidden" / "binding" / f"xr_{fam}_test.go")
        print(f"{fam}: artifacts written")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
