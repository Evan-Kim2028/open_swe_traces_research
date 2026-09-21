#!/usr/bin/env python3
"""Write _author artifacts for the 6 kops->deps-modules xrepo20 units.

Same layout as the gin author script; hidden tests live in
/tmp/xrh/xr_<fam>_test.go (verified pristine-passing in ladder-base:kops).
Cheats here patch the consumer call sites in pkg/apis/kops/validation and add
a shallow fallback appended to the edited consumer file — never touching deps/.
"""
from __future__ import annotations

import argparse
import difflib
import shutil
from pathlib import Path

ROOT = Path("/home/evan/Documents/oswt-XREPO20")
AUTHOR = ROOT / "experiments/pipeline/authored_batch2/kops"
HIDDEN_SRC = Path("/tmp/xrh")
KV = "pkg/apis/kops/validation"
PSRC = ROOT / "experiments/xrepo20/pair/kops/src"

REPRO = """Reproduce with:

```
tests/test.sh
```

(which copies the hidden suite into the tree and runs
`go test -count=1 -timeout 15m ./pkg/apis/kops/validation/...`)

Please identify and fix the underlying logic bug so these tests pass. Do not
skip, delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break existing
behavior.
"""


def _diff(old: str, new: str, rel: str) -> str:
    return "".join(difflib.unified_diff(
        old.splitlines(keepends=True), new.splitlines(keepends=True),
        fromfile=f"a/{rel}", tofile=f"b/{rel}"))


def _newfile(path: str, content: str) -> str:
    lines = content.splitlines(keepends=True)
    return f"--- /dev/null\n+++ b/{path}\n@@ -0,0 +1,{len(lines)} @@\n" + "".join("+" + l for l in lines)


def api(fam, surface):
    return (f"# Exported API — {fam}\n\n"
            f"Consumer-facing: `{surface}`. The hidden suite only touches\n"
            "exported consumer API (`pkg/apis/kops/validation`); the excised\n"
            "functions live in the pinned dependency modules under `deps/`.\n")


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
            "- Consumer C: `example.internal/kops` (kOps; dep modules via `replace`)\n"
            "- Library L: `k8s.io/apimachinery` / `github.com/blang/semver/v4` staged as nested modules under `deps/`.\n"
            f"- Excised funcs: {funcs}\n"
            f"- Kept in place: {kept}\n"
            f"- Reachability: {reach}\n")


def difficulty(fam, body, items=()):
    d = "\n".join(f"  {i+1}. {x}" for i, x in enumerate(items))
    return f"# Why hard — {fam}\n\npredicted_flip: L2\ndetails:\n{d}\n\n{body}\n"


UNITS = {}

UNITS["ignames"] = dict(
    api=api("ignames", "`ValidateInstanceGroupName(name, fieldPath)`"),
    bug=bugreport(
        "Instance-group name validation crashes. Calling\n"
        "`ValidateInstanceGroupName` on any name — valid or invalid — aborts\n"
        "with a `panic:` instead of returning a field.ErrorList. Other\n"
        "instance-group validation paths still work."),
    contract=contract("instance-group name validation", """Symptom: `ValidateInstanceGroupName` aborts with a `panic:` on any input.

`ValidateInstanceGroupName(name, field.NewPath("metadata", "name"))` must
validate the name as a DNS-1123 subdomain and report violations as
`field.Error`s at `metadata.name`:

- Valid: lowercase alphanumeric + `-` + `.` labels, total length ≤ 253.
  `nodes`, `node-1`, `a`, `a-b.c-d`, `1nodes` (leading digit allowed),
  253-char names — all pass.
- Rejected: empty, uppercase anywhere, `_`, leading/trailing `-` or `.`,
  empty mid-labels (`no..des`), spaces, >253 chars.
- Quirk: per-label 63-char limits are NOT enforced at this level (a 64-char
  single label passes).
- Every error's `Field` is exactly `metadata.name`.""",
        [("TestXRIGNames", "name corpus: valid DNS-1123 subdomains pass, violations rejected at metadata.name")]),
    details=[
        "Accepts DNS-1123 subdomains: lowercase alnum + `-` + `.`, ≤253 chars.",
        "Leading digits allowed (`1nodes`); uppercase, `_`, spaces rejected.",
        "Empty, leading/trailing `-`/`.`, empty mid-labels, >253 chars rejected.",
        "Per-label 63-char limit NOT enforced (64-char label passes).",
        "Errors land at `metadata.name`.",
    ],
    difficulty="""The excised layer is a thin wrapper — `NameIsDNSSubdomain` in
`pkg/api/validation/generic.go` — but the panic only surfaces through the
consumer's `ValidateInstanceGroupName`, and the solver must find the
`replace`-wired module boundary. The contract quirks (leading digits OK, per-label length NOT
checked, total ≤253) pin the exact DNS-1123-subdomain semantics.""",
    closure=("`NameIsDNSSubdomain`, `NameIsDNSLabel`, `NameIsDNS1035Label`, `maskTrailingDash` (4 funcs, `deps/apimachinery/pkg/api/validation/generic.go`).",
             "`util/validation.IsDNS1123Subdomain` (the delegate) intact — it belongs to the clusternames unit's closure; `ValidateObjectMeta` intact.",
             "`ValidateInstanceGroupName` → `NameIsDNSSubdomain(name, false)`; only NameIsDNSSubdomain is exercised by the suite (prefix-mode siblings are dead-by-design)."),
    cheat=None,  # set below
)

UNITS["clusternames"] = dict(
    api=api("clusternames", "`ValidateCluster(cluster, strict, vfsContext)` — `objectMeta.name` path"),
    bug=bugreport(
        "Cluster name validation is broken. `ValidateCluster` on a spec whose\n"
        "name violates DNS-1123 subdomain rules no longer reports an\n"
        "`objectMeta.name` error — invalid names sail through, or validation\n"
        "aborts with a `panic:`. Other spec checks still run."),
    contract=contract("cluster name validation", """Symptom: `ValidateCluster` either misses invalid cluster names or aborts with a `panic:`.

`ValidateCluster` must validate `cluster.ObjectMeta.Name` and report
violations at field path `objectMeta.name`:

- The name must be a DNS-1123 subdomain: lowercase alphanumeric + `-` + `.`
  labels, ≤253 chars. `cluster.example.com`, `a.b`, `my-cluster.example.com`
  pass; `Bad_Name.example.com`, `CLUSTER.EXAMPLE.COM`, `cluster..example.com`,
  `-bad.example.com`, `bad-.example.com`, `cluster.example.com.` (trailing
  dot → empty label), >253 chars — all rejected.
- Additionally a single-label name (`singlelabel`, no dot) is rejected as not
  fully-qualified unless the cluster uses None-DNS topology.
- Errors land at `objectMeta.name` (the objectMeta path), distinct from the
  `metadata.name` path used by generic object-meta validation.""",
        [("TestXRClusterNames", "name corpus: DNS-1123 + fqdn-dot rule enforced at objectMeta.name")]),
    details=[
        "Cluster names must be DNS-1123 subdomains: lowercase alnum + `-` + `.`, ≤253.",
        "Uppercase, `_`, empty labels, leading/trailing dashes/dots rejected.",
        "Single-label names rejected as non-FQDN (None-DNS clusters exempt).",
        "Violations land at `objectMeta.name`.",
    ],
    difficulty="""`ValidateCluster` checks the name TWICE through two different
dependency functions (`ValidateObjectMeta` → `metadata.name`, then
`IsDNS1123Subdomain` → `objectMeta.name`). Only the second is excised — the
solver must realise the two paths are independent and that the suite asserts
on `objectMeta.name` specifically. The FQDN-dot rule on top of DNS-1123 adds
a consumer-side clause the dep doesn't cover.""",
    closure=("`IsDNS1123Subdomain`, `IsDNS1123Label`, `IsDNS1035Label`, `IsWildcardDNS1123Subdomain`, `IsDNS1123SubdomainWithUnderscore` (5 funcs, `deps/apimachinery/pkg/util/validation/validation.go`).",
             "`NameIsDNSSubdomain`/`ValidateObjectMeta` (generic.go) intact — the `metadata.name` check still runs; only the `objectMeta.name` path is excised.",
             "`ValidateCluster` → `newValidateCluster` → `IsDNS1123Subdomain(clusterName)` at the `objectMeta.name` path."),
    cheat=None,
)

UNITS["labelvals"] = dict(
    api=api("labelvals", "`CrossValidateInstanceGroup(ig, cluster, cloud, strict)` — karpenter `excludedInstanceTypes`"),
    bug=bugreport(
        "Karpenter instance-requirement validation is broken. For a\n"
        "Karpenter-managed node InstanceGroup, entries in\n"
        "`spec.mixedInstancesPolicy.instanceRequirements.excludedInstanceTypes`\n"
        "that violate label-value rules are accepted silently — or\n"
        "`CrossValidateInstanceGroup` aborts with a `panic:`. Other\n"
        "instance-group checks still run."),
    contract=contract("karpenter excludedInstanceTypes validation", """Symptom: `CrossValidateInstanceGroup` accepts invalid excludedInstanceTypes entries or aborts with a `panic:`.

For a Karpenter-managed Node InstanceGroup on an AWS cluster with
`spec.karpenter.enabled`, each entry of
`spec.mixedInstancesPolicy.instanceRequirements.excludedInstanceTypes` must be
validated:

- Empty entries rejected with `must not be empty`.
- `<family>.*` wildcard entries are unwrapped and the family name validated.
- Any other `*` in the entry is rejected outright.
- The (unwrapped) entry must be a valid Kubernetes LABEL VALUE: ≤63 chars
  TOTAL, `[A-Za-z0-9_.-]` charset, starting AND ending alphanumeric. Uppercase
  IS allowed (this is not DNS); `m5..large` (consecutive dots mid-string) is
  valid; `-m5.large`, `m5.large-`, `café.large`, 64+ chars all fail.
- Violations land at
  `spec.mixedInstancesPolicy.instanceRequirements.excludedInstanceTypes[i]`
  with the entry index.""",
        [
            ("TestXRKarpenterExcludedTypes", "entry corpus: label-value charset/length/endpoints + wildcard rules"),
            ("TestXRKarpenterExcludedIndex", "errors carry the entry index"),
        ]),
    details=[
        "Each `excludedInstanceTypes` entry must be a valid label value: ≤63 chars, `[A-Za-z0-9_.-]`, alnum start+end.",
        "Uppercase and mid-string dots/underscores are valid label values.",
        "`<family>.*` wildcards unwrap before validation; other `*` uses rejected.",
        "Empty entries rejected before label validation.",
        "Errors land at `…excludedInstanceTypes[i]` with the index.",
    ],
    difficulty="""The seam is three hops deep: `CrossValidateInstanceGroup` →
`validateKarpenterInstanceGroup` → `validateKarpenterInstanceRequirements` →
`IsLabelValue`. The fixture alone is non-trivial (Karpenter manager + AWS +
karpenter.enabled + MixedInstancesPolicy). The label-value grammar inverts
DNS intuition: uppercase and consecutive dots are FINE. Reaching the excised
function requires all the consumer gates to pass first.""",
    closure=("`IsLabelValue`, `IsLabelKey`, `IsPrefixedLabelKey`, `prefixEach` (4 funcs, `deps/apimachinery/pkg/api/validate/content/kube.go`).",
             "the `util/validation` DNS helpers (clusternames' closure) intact; karpenter requirement plumbing intact.",
             "`CrossValidateInstanceGroup` → `validateKarpenterInstanceGroup` → `validateKarpenterInstanceRequirements` → `IsLabelValue` on unwrapped wildcard entries."),
    cheat=None,
)

UNITS["portrange"] = dict(
    api=api("portrange", "`ValidateCluster` — `spec.kubeAPIServer.serviceNodePortRange`"),
    bug=bugreport(
        "API-server port-range validation is broken. `ValidateCluster` accepts\n"
        "malformed `spec.kubeAPIServer.serviceNodePortRange` values — or\n"
        "aborts with a `panic:` — instead of reporting an error at\n"
        "`spec.kubeAPIServer.serviceNodePortRange`."),
    contract=contract("service node port range", """Symptom: `ValidateCluster` misses invalid port ranges or aborts with a `panic:`.

`spec.kubeAPIServer.serviceNodePortRange` must parse as a port range:

- `LOW-HIGH` hyphen form, `LOW+COUNT` plus form, or a single port `N`.
- Outer whitespace is trimmed; interior whitespace breaks the parse.
- Both forms reject when either side is non-numeric, when `high < low`, or
  when either port exceeds 65535. Port 0 IS allowed; there is no minimum.
- A single integer is a one-port range (`443` valid).
- Mixed `+`/`-` notation and extra separators reject.
- An empty string is not validated (the consumer skips it).
- Errors land at `spec.kubeAPIServer.serviceNodePortRange`.""",
        [("TestXRServiceNodePortRange", "range corpus: hyphen/plus/single notations, bounds, whitespace, mixed")]),
    details=[
        "Accepts `LOW-HIGH`, `LOW+COUNT`, and single-port forms; outer whitespace trimmed.",
        "Rejects reversed ranges, ports >65535, non-numeric, interior whitespace, mixed notation.",
        "Port 0 allowed; single int = one-port range.",
        "Errors land at `spec.kubeAPIServer.serviceNodePortRange`.",
    ],
    difficulty="""`PortRange.Set` hides three notations (hyphen, plus, single)
behind a bitmask parse — a solver who only handles `a-b` fails `30000+100`
and `443`. The quirks stack: TrimSpace happens BEFORE splitting, port 0 is
legal, and mixed `+`/`-` is a distinct reject path. All inside
`deps/apimachinery/pkg/util/net`.""",
    closure=("`Set`, `ParsePortRange`, `ParsePortRangeOrDie`, `Contains`, `String`, `Type` (6 funcs, `deps/apimachinery/pkg/util/net/port_range.go`).",
             "other util/net helpers intact; `validateKubeAPIServer` plumbing intact.",
             "`ValidateCluster` → `newValidateCluster` → `validateClusterSpec` → `validateKubeAPIServer` → `PortRange.Set`."),
    cheat=None,
)

UNITS["intstrpct"] = dict(
    api=api("intstrpct", "`ValidateInstanceGroup` — `spec.rollingUpdate.maxUnavailable`/`maxSurge`"),
    bug=bugreport(
        "Rolling-update bound validation is broken. `ValidateInstanceGroup`\n"
        "accepts unparseable `maxUnavailable`/`maxSurge` values — or aborts\n"
        "with a `panic:` — instead of reporting errors at\n"
        "`spec.rollingUpdate.*`. The zero-zero combination and control-plane\n"
        "surge rules are also affected."),
    contract=contract("rolling-update maxUnavailable/maxSurge", """Symptom: `ValidateInstanceGroup` misses invalid rolling-update bounds or aborts with a `panic:`.

`spec.rollingUpdate.maxUnavailable` and `.maxSurge` are `intstr` values —
either an absolute int or a `N%` percent string scaled against a total:

- Int values pass through; percent strings scale: `maxUnavailable` uses
  total=1 rounding DOWN (`25%`→0, `50%`→0), `maxSurge` uses total=1000
  rounding UP (`25%`→250).
- A plain numeric string WITHOUT `%` (`"5"`) parses as absolute — valid.
  Non-numeric strings (`"abc%"`, `"five"`, `"x%"`) fail with `Unable to parse`.
- Negative results produce `Cannot be negative`; there is NO upper bound
  (`150%` is allowed).
- `unavailable == 0 && surge == 0` (after scaling) is `Forbidden` at
  `spec.rollingUpdate.maxSurge`.
- On control-plane InstanceGroups any nonzero surge is `Forbidden`.
- Errors land at `spec.rollingUpdate.maxUnavailable` / `.maxSurge`.""",
        [
            ("TestXRRollingUpdateParse", "intstr corpus: int/percent/plain-string parses, negatives, zero-zero, cp-surge"),
            ("TestXRRollingUpdateRoundDown", "percent-of-1 rounds down to 0 — the scaled-value edge"),
        ]),
    details=[
        "Int values pass through; `N%` scales (down for maxUnavailable/1, up for maxSurge/1000).",
        "Plain numeric strings parse as absolute; non-numeric fail with `Unable to parse`.",
        "Negatives rejected; no upper bound.",
        "Both-zero after scaling → Forbidden at maxSurge.",
        "Control-plane IGs forbid nonzero surge.",
        "Errors land at `spec.rollingUpdate.maxUnavailable`/`.maxSurge`.",
    ],
    difficulty="""The semantics are in `GetScaledValueFromIntOrPercent` — three
layers deep in `deps/`. Traps: a bare `"5"` (no %) is an absolute value, not an
error; `25%` of total 1 rounds DOWN to 0, which then trips the both-zero
Forbidden — the scaled value, not the literal, drives the rule; and `150%`
is legal. The solver must reproduce ceil/floor asymmetry across two call
sites with different totals.""",
    closure=("`GetScaledValueFromIntOrPercent`, `GetValueFromIntOrPercent`, `getIntOrPercentValue`, `getIntOrPercentValueSafely` (4 funcs, `deps/apimachinery/pkg/util/intstr/intstr.go`).",
             "the `IntOrString` type, `FromInt`/`FromString`/`Parse` constructors intact.",
             "`ValidateInstanceGroup` → `validateRollingUpdate` → `GetScaledValueFromIntOrPercent` at both fields."),
    cheat=None,
)

UNITS["semververs"] = dict(
    api=api("semververs", "`ValidateCluster` — `spec.kubernetesVersion`, `networking.cilium.version`, `containerd.version`, `etcdClusters[].version`, `containerd.nri`"),
    bug=bugreport(
        "Version-string validation is broken. `ValidateCluster` accepts\n"
        "malformed version fields — or aborts with a `panic:` — instead of\n"
        "reporting errors at the `spec.*.version` paths. This affects\n"
        "`kubernetesVersion`, `networking.cilium.version`,\n"
        "`containerd.version` and `etcdClusters[].version`."),
    contract=contract("version-string validation", """Symptom: `ValidateCluster` misses invalid versions or aborts with a `panic:`.

Several spec fields hold version strings validated by strict or tolerant
semantic-version parsing:

- `spec.kubernetesVersion` — TOLERANT parse (`v`-prefix, missing patch, and
  prerelease/build suffixes all fine: `1.30.0`, `v1.30.0`, `1.30`,
  `1.30.0-alpha.1`, `1.30.0+build`); unparseable → `Invalid` at
  `spec.kubernetesVersion` and validation returns early.
- `spec.networking.cilium.version` — consumer requires a `v` prefix; the
  remainder must STRICTLY parse (`v1.18` fails — missing patch); only minor
  version 18 is supported (`v1.17.0`, `v1.19.0`, `v2.0.0` rejected);
  `v1.18.0-rc.1` parses and passes.
- `spec.containerd.version` — TOLERANT parse; must be ≥2.1.0 (`1.7.0`,
  `2.0.9`, `2.1.0-rc.1` rejected — a prerelease is below its release).
- `spec.etcdClusters[].version` — `v`-prefix trimmed by the consumer, then
  STRICT parse; only major version 3 supported.
- `spec.containerd` with NRI enabled additionally runs a version-RANGE check
  (`>=1.7.0`) — reachable but subsumed by the ≥2.1.0 gate.
- Errors land at `spec.kubernetesVersion`, `spec.networking.cilium.version`,
  `spec.containerd.version`, `spec.etcdClusters[i].version`.""",
        [
            ("TestXRKubernetesVersionParse", "tolerant k8sVersion parse; unparseable → early Invalid"),
            ("TestXRCiliumVersion", "v-prefix + strict parse + minor==18 gate"),
            ("TestXRContainerdVersion", "tolerant parse + >=2.1.0 gate; prerelease ordering"),
            ("TestXREtcdVersion", "v-trim + strict parse + major==3 gate"),
            ("TestXRNRIVersionRange", "NRI-enabled path exercises the range machinery"),
        ]),
    details=[
        "`spec.kubernetesVersion` uses tolerant parse (v-prefix, missing patch, pre/build ok); failure returns early.",
        "`spec.networking.cilium.version` requires `v` prefix, STRICT parse, minor==18.",
        "`spec.containerd.version` uses tolerant parse and requires >=2.1.0 (prereleases order below releases).",
        "`spec.etcdClusters[].version` trims `v`, STRICT parses, requires major==3.",
        "NRI-enabled containerd exercises a `>=1.7.0` range check (subsumed by the 2.1.0 gate).",
        "Errors land at the four `spec.*.version` paths.",
    ],
    difficulty="""One dep package (`blang/semver`) with two parse modes —
`Parse` strict vs `ParseTolerant` — consumed at FIVE different call sites
with different surrounding gates. The solver must map which site is strict
(cilium, etcd) vs tolerant (k8sVersion, containerd), get prerelease ordering
right (`2.1.0-rc.1 < 2.1.0`), and implement `ParseRange` for the NRI path.
The early-return on k8sVersion failure is a separate behavioral commitment.""",
    closure=("`Parse`, `ParseTolerant` (`deps/semver/semver.go`); `ParseRange` (`deps/semver/range.go`).",
             "`semver.Version`, `MustParse`, `Version.LT/GT`, `VersionRange` machinery intact — only the three parsers excised.",
             "`ValidateCluster` → `util.ParseKubernetesVersion` (→`ParseTolerant`), `validateNetworkingCilium` (→`Parse`), `validateContainerdConfig` (→`ParseTolerant`), `validateNriConfig` (→`Parse`+`ParseRange`), `validateEtcdVersion` (→`Parse`)."),
    cheat=None,
)


# --- cheat patches: consumer call-site bypass + shallow fallback ------------
# The shallow helper is appended to the SAME file that loses the dep call, so
# (a) the file keeps a func symbol for the overlap checker's disjointness
# evidence, and (b) imports orphaned by the swap are dropped in the same diff.

def cheats() -> dict[str, str]:
    ig = (PSRC / KV / "instancegroup.go").read_text()
    val = (PSRC / KV / "validation.go").read_text()
    leg = (PSRC / KV / "legacy.go").read_text()
    out = {}

    def swap(text: str, pairs: list[tuple[str, str]], drop_imports: list[str],
             helper: str, add_imports: list[str] | None = None) -> str:
        new = text
        for old, rep in pairs:
            assert old in new, f"missing call site: {old}"
            new = new.replace(old, rep, 1)
        for imp in drop_imports:
            line = f"\t{imp}\n"
            assert line in new, f"missing import line: {imp}"
            new = new.replace(line, "", 1)
        for imp in add_imports or []:
            anchor = '\t"strings"\n'
            assert anchor in new, "strings import anchor missing"
            new = new.replace(anchor, f'\t{imp}\n{anchor}', 1)
        return new + helper

    # ignames: replace the dep call in ValidateInstanceGroupName with a
    # permissive regexp check that misses charset/endpoint edges.
    out["ignames"] = _diff(ig, swap(ig, [
        ("\tfor _, msg := range apivalidation.NameIsDNSSubdomain(name, false) {",
         "\tfor _, msg := range xrIGNameCheck(name) {"),
    ], ["apivalidation \"k8s.io/apimachinery/pkg/api/validation\""], '''
// xrIGNameCheck is a shallow local replacement — it misses the empty-label,
// endpoint-dash and per-label rules entirely.
func xrIGNameCheck(name string) []string {
	if name == "" || len(name) > 253 || !xrIGNameRe.MatchString(name) {
		return []string{"must be a DNS-1123 subdomain"}
	}
	return nil
}

var xrIGNameRe = regexp.MustCompile(`^[a-z0-9.-]+$`)
'''), f"{KV}/instancegroup.go")

    # clusternames: bypass the IsDNS1123Subdomain call in newValidateCluster
    out["clusternames"] = _diff(val, swap(val, [
        ("\t\terrs := utilvalidation.IsDNS1123Subdomain(clusterName)",
         "\t\terrs := xrCNNameCheck(clusterName)"),
    ], ["utilvalidation \"k8s.io/apimachinery/pkg/util/validation\""], '''
// xrCNNameCheck is a shallow local replacement — it only checks for a dot
// and lowercase, missing the label grammar entirely.
func xrCNNameCheck(name string) []string {
	if !strings.Contains(name, ".") || name != strings.ToLower(name) {
		return []string{"must be a DNS-1123 subdomain"}
	}
	return nil
}
'''), f"{KV}/validation.go")

    # labelvals: bypass the IsLabelValue call in validateKarpenterInstanceRequirements
    out["labelvals"] = _diff(ig, swap(ig, [
        ("\t\tfor _, msg := range contentvalidation.IsLabelValue(value) {",
         "\t\tfor _, msg := range xrLVLabelValue(value) {"),
    ], ["contentvalidation \"k8s.io/apimachinery/pkg/api/validate/content\""], '''
var xrLVRe = regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)

// xrLVLabelValue is a shallow local replacement — it misses the 63-char cap,
// underscores, and the alnum start/end rules.
func xrLVLabelValue(value string) []string {
	if len(value) > 63 || !xrLVRe.MatchString(value) {
		return []string{"must be a valid label value"}
	}
	return nil
}
'''), f"{KV}/instancegroup.go")

    # portrange: bypass pr.Set in validateKubeAPIServer
    out["portrange"] = _diff(val, swap(val, [
        ("\t\tpr := &utilnet.PortRange{}\n\t\terr := pr.Set(v.ServiceNodePortRange)",
         "\t\terr := xrPRSet(v.ServiceNodePortRange)"),
    ], ["utilnet \"k8s.io/apimachinery/pkg/util/net\""], add_imports=['"strconv"'], helper='''
// xrPRSet is a shallow local replacement — it handles only the hyphen form
// and misses plus-notation, single ports, trimming and the bounds checks.
func xrPRSet(value string) error {
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return fmt.Errorf("unable to parse port range: %s", value)
	}
	if _, err := strconv.Atoi(parts[0]); err != nil {
		return err
	}
	if _, err := strconv.Atoi(parts[1]); err != nil {
		return err
	}
	return nil
}
'''), f"{KV}/validation.go")

    # intstrpct: bypass GetScaledValueFromIntOrPercent at both rolling-update
    # sites; the helper's *intstr.IntOrString signature keeps the import live.
    out["intstrpct"] = _diff(val, swap(val, [
        ("\t\tunavailable, err = intstr.GetScaledValueFromIntOrPercent(rollingUpdate.MaxUnavailable, 1, false)",
         "\t\tunavailable, err = xrIPScaled(rollingUpdate.MaxUnavailable, 1, false)"),
        ("\t\tsurge, err := intstr.GetScaledValueFromIntOrPercent(rollingUpdate.MaxSurge, 1000, true)",
         "\t\tsurge, err := xrIPScaled(rollingUpdate.MaxSurge, 1000, true)"),
    ], [], add_imports=['"strconv"'], helper='''
// xrIPScaled is a shallow local replacement — it drops the strconv error,
// so a non-numeric percent silently scales to 0, and it ignores roundUp.
func xrIPScaled(v *intstr.IntOrString, total int, roundUp bool) (int, error) {
	if v.Type == intstr.Int {
		return int(v.IntVal), nil
	}
	s := v.StrVal
	if !strings.HasSuffix(s, "%") {
		return 0, fmt.Errorf("invalid value for IntOrString: %v", s)
	}
	pct, _ := strconv.Atoi(strings.TrimSuffix(s, "%"))
	return pct * total / 100, nil
}
'''), f"{KV}/validation.go")

    # semververs: bypass all five call sites with a naive sscanf parse. The
    # helpers append to validation.go; legacy.go loses its lone `util` use.
    new_val = swap(val, [
        ("\t\tversion, err := semver.Parse(versionString)",
         "\t\tversion, err := xrSVParse(versionString)"),
        ("\t\tsv, err := semver.ParseTolerant(*config.Version)",
         "\t\tsv, err := xrSVParse(*config.Version)"),
        ("\tv, err := semver.Parse(*containerd.Version)",
         "\tv, err := xrSVParse(*containerd.Version)"),
        ("\texpectedRange, err := semver.ParseRange(\">=1.7.0\")",
         "\texpectedRange, err := xrSVRange(\">=1.7.0\")"),
        ("\tsem, err := semver.Parse(strings.TrimPrefix(version, \"v\"))",
         "\tsem, err := xrSVParse(strings.TrimPrefix(version, \"v\"))"),
    ], [], add_imports=['"strconv"'], helper='''
// xrSVParse is a shallow local replacement — sscanf %d.%d.%d only; no
// prerelease, build, v-prefix or tolerant handling.
func xrSVParse(s string) (semver.Version, error) {
	var v semver.Version
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, fmt.Errorf("unable to parse version %q", s)
	}
	var err error
	nums := [3]*uint64{&v.Major, &v.Minor, &v.Patch}
	for i, p := range parts {
		n, e := strconv.ParseUint(p, 10, 64)
		if e != nil {
			return v, e
		}
		*nums[i] = n
		err = e
	}
	return v, err
}

func xrSVRange(s string) (semver.Range, error) {
	return func(semver.Version) bool { return true }, nil // accepts everything
}

func xrSVK8s(s string) (*semver.Version, error) {
	v, err := xrSVParse(strings.TrimPrefix(s, "v"))
	if err != nil {
		return nil, err
	}
	return &v, nil
}
''')
    new_leg = swap(leg, [
        ("\t\tk8sVersion, err = util.ParseKubernetesVersion(c.Spec.KubernetesVersion)",
         "\t\tk8sVersion, err = xrSVK8s(c.Spec.KubernetesVersion)"),
    ], ["\"example.internal/kops/pkg/apis/kops/util\""], "")
    out["semververs"] = _diff(val, new_val, f"{KV}/validation.go") + _diff(leg, new_leg, f"{KV}/legacy.go")
    return out


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--hidden-src", type=Path, default=HIDDEN_SRC)
    ap.add_argument("--families", nargs="*", default=None)
    args = ap.parse_args()
    cheat_map = cheats()
    fams = args.families or sorted(UNITS)
    for fam in fams:
        spec = UNITS[fam]
        a = AUTHOR / fam / "_author"
        (a / "hidden" / KV).mkdir(parents=True, exist_ok=True)
        (a / "api.md").write_text(spec["api"], encoding="utf-8")
        (a / "bugreport.md").write_text(spec["bug"], encoding="utf-8")
        (a / "contract.md").write_text(spec["contract"], encoding="utf-8")
        (a / "closure.md").write_text(closure(fam, *spec["closure"]), encoding="utf-8")
        (a / "difficulty.md").write_text(difficulty(fam, spec["difficulty"], spec["details"]), encoding="utf-8")
        (a / "DETAILS.md").write_text(details(spec["details"]), encoding="utf-8")
        (a / "cheat.patch").write_text(cheat_map[fam], encoding="utf-8")
        hidden = args.hidden_src / f"xr_{fam}_test.go"
        shutil.copy2(hidden, a / "hidden" / KV / f"xr_{fam}_test.go")
        print(f"{fam}: artifacts written")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
