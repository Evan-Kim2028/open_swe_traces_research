// Hidden black-box property suite for the ldapdn unit.
// Drives only the exported API in internal/ldap (api.md): FromCertSubject,
// FromRawCertSubject, ParseDN, DN.Equal/RDNsMatch/AncestorOf,
// RelativeDN.Equal, AttributeTypeAndValue.Equal.
// One property per DETAILS.md line; seeded random inputs, no worked examples.
package ldap_test

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"

	ldap "example.internal/msgkit/v2/internal/ldap"
)

const bbDnHiddenSeed = 20260919

func bbDnSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return bbDnHiddenSeed
}

func bbDnRng(t *testing.T) *rand.Rand {
	t.Helper()
	return rand.New(rand.NewSource(bbDnSeed()))
}

var bbDnTypes = []string{"CN", "OU", "O", "C", "L", "ST", "STREET", "POSTALCODE", "SERIALNUMBER", "DC"}

var bbDnOIDs = map[string][]int{
	"CN":           {2, 5, 4, 3},
	"SERIALNUMBER": {2, 5, 4, 5},
	"C":            {2, 5, 4, 6},
	"L":            {2, 5, 4, 7},
	"ST":           {2, 5, 4, 8},
	"STREET":       {2, 5, 4, 9},
	"O":            {2, 5, 4, 10},
	"OU":           {2, 5, 4, 11},
	"POSTALCODE":   {2, 5, 4, 17},
	"DC":           {0, 9, 2342, 19200300, 100, 1, 25},
}

func bbDnRandValue(rng *rand.Rand) string {
	const alpha = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 -_"
	n := 1 + rng.Intn(10)
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteByte(alpha[rng.Intn(len(alpha))])
	}
	return sb.String()
}

func bbDnRandDN(rng *rand.Rand) *ldap.DN {
	nR := 1 + rng.Intn(4)
	dn := &ldap.DN{}
	for i := 0; i < nR; i++ {
		rdn := &ldap.RelativeDN{}
		nA := 1 + rng.Intn(3)
		for j := 0; j < nA; j++ {
			rdn.Attributes = append(rdn.Attributes, &ldap.AttributeTypeAndValue{
				Type:  bbDnTypes[rng.Intn(len(bbDnTypes))],
				Value: bbDnRandValue(rng),
			})
		}
		dn.RDNs = append(dn.RDNs, rdn)
	}
	return dn
}

// bbDnRender serializes a DN back into RFC4514 form, escaping specials.
func bbDnRender(dn *ldap.DN) string {
	var parts []string
	for _, rdn := range dn.RDNs {
		var attrs []string
		for _, a := range rdn.Attributes {
			var v strings.Builder
			for i := 0; i < len(a.Value); i++ {
				c := a.Value[i]
				if strings.IndexByte(",+;<>\"\\=", c) >= 0 || (i == 0 && c == ' ') || (i == len(a.Value)-1 && c == ' ') {
					v.WriteByte('\\')
				}
				v.WriteByte(c)
			}
			attrs = append(attrs, a.Type+"="+v.String())
		}
		parts = append(parts, strings.Join(attrs, "+"))
	}
	return strings.Join(parts, ",")
}

func bbDnCountAttrs(dn *ldap.DN) int {
	n := 0
	for _, r := range dn.RDNs {
		n += len(r.Attributes)
	}
	return n
}

// Detail 1: ',' separates relative names, '+' separates attribute pairs
// within one relative name.
func TestDetail01_CommaAndPlusSeparators(t *testing.T) {
	rng := bbDnRng(t)
	for iter := 0; iter < 60; iter++ {
		want := bbDnRandDN(rng)
		str := bbDnRender(want)
		got, err := ldap.ParseDN(str)
		if err != nil {
			t.Fatalf("iter %d: ParseDN(%q) err=%v", iter, str, err)
		}
		if len(got.RDNs) != len(want.RDNs) {
			t.Fatalf("iter %d: %q -> %d RDNs, want %d", iter, str, len(got.RDNs), len(want.RDNs))
		}
		for i, r := range want.RDNs {
			if len(got.RDNs[i].Attributes) != len(r.Attributes) {
				t.Fatalf("iter %d rdn %d: %q -> %d attrs, want %d", iter, i, str, len(got.RDNs[i].Attributes), len(r.Attributes))
			}
			for j, a := range r.Attributes {
				ga := got.RDNs[i].Attributes[j]
				if ga.Type != a.Type || ga.Value != a.Value {
					t.Fatalf("iter %d rdn %d attr %d: %q -> %s=%s, want %s=%s", iter, i, j, str, ga.Type, ga.Value, a.Type, a.Value)
				}
			}
		}
	}
	// Hand-rolled shapes: multi-attr RDNs via '+' never split into RDNs.
	got, err := ldap.ParseDN("a=1+b=2+c=3,d=4")
	if err != nil || len(got.RDNs) != 2 || len(got.RDNs[0].Attributes) != 3 || len(got.RDNs[1].Attributes) != 1 {
		t.Fatalf("multi-attr rdn shape: %+v err=%v", got, err)
	}
}

// Detail 2: unescaped leading spaces skipped; unescaped trailing spaces
// before '=', ',', '+' or EOF trimmed; interior spaces kept.
func TestDetail02_SpaceHandling(t *testing.T) {
	rng := bbDnRng(t)
	for iter := 0; iter < 50; iter++ {
		typ := bbDnTypes[rng.Intn(len(bbDnTypes))]
		val := bbDnRandValue(rng)
		// Pad the type and value with random unescaped lead/trail spaces.
		// Value is pre-trimmed so only the padding exercises the trim rules;
		// no spaces are placed between '=' and the value.
		val = strings.TrimSpace(val)
		if val == "" {
			val = "x"
		}
		str := strings.Repeat(" ", rng.Intn(4)) + typ + strings.Repeat(" ", rng.Intn(3)) +
			"=" + val + strings.Repeat(" ", rng.Intn(4))
		got, err := ldap.ParseDN(str)
		if err != nil {
			t.Fatalf("iter %d: ParseDN(%q) err=%v", iter, str, err)
		}
		a := got.RDNs[0].Attributes[0]
		if a.Type != typ || a.Value != val {
			t.Fatalf("iter %d: %q -> (%q,%q) want (%q,%q)", iter, str, a.Type, a.Value, typ, val)
		}
	}
	// Interior spaces survive.
	got, err := ldap.ParseDN("cn=a  b   c")
	if err != nil || got.RDNs[0].Attributes[0].Value != "a  b   c" {
		t.Fatalf("interior spaces: %+v err=%v", got, err)
	}
	// Spaces before ',' are trimmed per RDN.
	got, err = ldap.ParseDN("cn=a   ,   ou=b ")
	if err != nil || len(got.RDNs) != 2 ||
		got.RDNs[0].Attributes[0].Value != "a" || got.RDNs[1].Attributes[0].Value != "b" {
		t.Fatalf("trailing spaces before comma: %+v err=%v", got, err)
	}
}

// Detail 3: escape rules — '\' + special is literal; '\' + 2 hex decodes a
// byte; exactly one char after '\' or bad hex errors; a trailing '\' is
// silently dropped.
func TestDetail03_Escapes(t *testing.T) {
	rng := bbDnRng(t)
	specials := []byte{' ', '"', '#', '+', ',', ';', '<', '=', '>', '\\'}
	for _, c := range specials {
		str := "cn=a\\" + string(c) + "z"
		got, err := ldap.ParseDN(str)
		if err != nil {
			t.Fatalf("ParseDN(%q) err=%v", str, err)
		}
		want := "a" + string(c) + "z"
		if got.RDNs[0].Attributes[0].Value != want {
			t.Fatalf("ParseDN(%q) value=%q want %q", str, got.RDNs[0].Attributes[0].Value, want)
		}
	}
	// Random hex escapes decode to the byte.
	for iter := 0; iter < 80; iter++ {
		b := byte(rng.Intn(256))
		str := "cn=x\\" + strings.ToUpper(strconv.FormatInt(int64(b), 16))
		if b < 16 {
			str = "cn=x\\0" + strings.ToUpper(strconv.FormatInt(int64(b), 16))
		}
		got, err := ldap.ParseDN(str)
		if err != nil {
			t.Fatalf("ParseDN(%q) err=%v", str, err)
		}
		want := "x" + string([]byte{b})
		if got.RDNs[0].Attributes[0].Value != want {
			t.Fatalf("ParseDN(%q) value=%q want %q", str, got.RDNs[0].Attributes[0].Value, want)
		}
	}
	// One char left after backslash -> error.
	if _, err := ldap.ParseDN("cn=x\\4"); err == nil {
		t.Fatal("cn=x\\4 should error (one char left)")
	}
	// Bad hex pair -> error.
	for _, bad := range []string{"cn=x\\4z", "cn=x\\zq", "cn=x\\g1"} {
		if _, err := ldap.ParseDN(bad); err == nil {
			t.Fatalf("%q should error (bad hex)", bad)
		}
	}
	// Backslash as final byte is dropped, no error.
	got, err := ldap.ParseDN("cn=x\\")
	if err != nil || got.RDNs[0].Attributes[0].Value != "x" {
		t.Fatalf("trailing backslash: %+v err=%v", got, err)
	}
}

// Detail 4: a value beginning '#' is rejected (unsupported BER).
func TestDetail04_HashValueRejected(t *testing.T) {
	for _, str := range []string{"cn=#dead", "cn=#41", "cn=#"} {
		if _, err := ldap.ParseDN(str); err == nil {
			t.Fatalf("%q should error (#-value unsupported)", str)
		}
	}
	// '#' mid-value or escaped is literal.
	got, err := ldap.ParseDN("cn=a#b")
	if err != nil || got.RDNs[0].Attributes[0].Value != "a#b" {
		t.Fatalf("mid-value '#': %+v err=%v", got, err)
	}
	got, err = ldap.ParseDN("cn=\\#abc")
	if err != nil || got.RDNs[0].Attributes[0].Value != "#abc" {
		t.Fatalf("escaped '#': %+v err=%v", got, err)
	}
}

// Detail 5: separator with no pending pair errors; EOF with buffered text
// and no '=' errors; EOF with pending type + empty value is silently
// dropped; empty input yields zero RDNs, no error.
func TestDetail05_EOFAndSeparatorEdges(t *testing.T) {
	for _, str := range []string{",", "+", ";", ",cn=a", "+cn=a", "cn=a,,cn=b", "cn=a,+,cn=b"} {
		if _, err := ldap.ParseDN(str); err == nil {
			t.Fatalf("%q should error (separator with no pending pair)", str)
		}
	}
	// ';' mid-value: this fork treats ';' as a literal value character, so
	// "cn=a;,cn=b" parses as [cn="a;"],[cn="b"]. If ';' were still a
	// separator the ',' would have no pending pair and this would error.
	got2, err := ldap.ParseDN("cn=a;,cn=b")
	if err != nil {
		t.Fatalf("cn=a;,cn=b err=%v", err)
	}
	if len(got2.RDNs) != 2 || got2.RDNs[0].Attributes[0].Value != "a;" {
		t.Fatalf("cn=a;,cn=b -> %+v want first RDN value \"a;\"", got2)
	}
	// Trailing separator at EOF is tolerated: "cn=a," parses the same as
	// "cn=a" (EOF after a separator has nothing buffered, so no pair to
	// demand). Mid-DN empty pairs still error (asserted above).
	got, err := ldap.ParseDN("cn=a,")
	if err != nil || len(got.RDNs) != 1 {
		t.Fatalf("cn=a, -> %+v err=%v", got, err)
	}
	// Buffered text, no '='.
	for _, str := range []string{"cn", "cn=a,x", "abc def"} {
		if _, err := ldap.ParseDN(str); err == nil {
			t.Fatalf("%q should error (no '=' before EOF)", str)
		}
	}
	// Pending type + empty value at EOF -> dropped.
	got, err = ldap.ParseDN("cn=")
	if err != nil || len(got.RDNs) != 0 {
		t.Fatalf("cn= -> %+v err=%v, want 0 RDNs", got, err)
	}
	got, err = ldap.ParseDN("cn=a=")
	if err != nil || len(got.RDNs) != 0 {
		t.Fatalf("cn=a= -> %+v err=%v, want 0 RDNs", got, err)
	}
	// Same drop mid-DN would still be an error at the separator; only EOF drops.
	got, err = ldap.ParseDN("")
	if err != nil || len(got.RDNs) != 0 {
		t.Fatalf("empty input -> %+v err=%v", got, err)
	}
	got, err = ldap.ParseDN("   ")
	if err != nil || len(got.RDNs) != 0 {
		t.Fatalf("spaces-only input -> %+v err=%v", got, err)
	}
}

// Detail 6: DN.Equal is positional — same count, pairwise equal in order.
func TestDetail06_PositionalDNEquality(t *testing.T) {
	rng := bbDnRng(t)
	for iter := 0; iter < 40; iter++ {
		a := bbDnRandDN(rng)
		b, err := ldap.ParseDN(bbDnRender(a))
		if err != nil {
			t.Fatalf("iter %d render-parse err=%v", iter, err)
		}
		if !a.Equal(b) {
			t.Fatalf("iter %d: identical DNs not Equal", iter)
		}
	}
	// Swap two RDNs -> not equal when >1 RDN.
	d1, _ := ldap.ParseDN("cn=a,ou=b")
	d2, _ := ldap.ParseDN("ou=b,cn=a")
	if d1.Equal(d2) || d2.Equal(d1) {
		t.Fatal("swapped RDN order should not be Equal")
	}
	// Count mismatch.
	d3, _ := ldap.ParseDN("cn=a")
	if d1.Equal(d3) || d3.Equal(d1) {
		t.Fatal("count mismatch should not be Equal")
	}
	// Same position, different value.
	d4, _ := ldap.ParseDN("cn=z,ou=b")
	if d1.Equal(d4) {
		t.Fatal("different value at same position should not be Equal")
	}
	// Empty vs empty equal.
	e1, _ := ldap.ParseDN("")
	e2, _ := ldap.ParseDN("")
	if !e1.Equal(e2) {
		t.Fatal("two empty DNs should be Equal")
	}
}

// Detail 7: RelativeDN.Equal is order-insensitive over attribute pairs;
// type case-insensitive; value case-sensitive.
func TestDetail07_RelativeDNSetEquality(t *testing.T) {
	r1, _ := ldap.ParseDN("cn=x+ou=y+o=z")
	r2, _ := ldap.ParseDN("o=z+cn=x+ou=y")
	if !r1.RDNs[0].Equal(r2.RDNs[0]) {
		t.Fatal("attribute order should not matter within an RDN")
	}
	// Type case-insensitive.
	r3, _ := ldap.ParseDN("CN=x")
	r4, _ := ldap.ParseDN("cn=x")
	if !r3.RDNs[0].Equal(r4.RDNs[0]) {
		t.Fatal("attribute type case should not matter")
	}
	// Value case-sensitive.
	r5, _ := ldap.ParseDN("cn=X")
	if r3.RDNs[0].Equal(r5.RDNs[0]) {
		t.Fatal("attribute value case must matter")
	}
	// Count mismatch.
	r6, _ := ldap.ParseDN("cn=x+ou=y")
	if r6.RDNs[0].Equal(r3.RDNs[0]) {
		t.Fatal("attribute count mismatch should not be Equal")
	}
	rng := bbDnRng(t)
	for iter := 0; iter < 40; iter++ {
		rdn := bbDnRandDN(rng).RDNs[0]
		// Shuffle attributes and render.
		attrs := append([]*ldap.AttributeTypeAndValue(nil), rdn.Attributes...)
		rng.Shuffle(len(attrs), func(i, j int) { attrs[i], attrs[j] = attrs[j], attrs[i] })
		got, err := ldap.ParseDN(bbDnRender(&ldap.DN{RDNs: []*ldap.RelativeDN{{Attributes: attrs}}}))
		if err != nil {
			t.Fatalf("iter %d err=%v", iter, err)
		}
		if !rdn.Equal(got.RDNs[0]) {
			t.Fatalf("iter %d: shuffled attrs not Equal", iter)
		}
	}
}

// Detail 8: RDNsMatch is a multiset match — duplicates need distinct
// counterparts.
func TestDetail08_RDNsMatchMultiset(t *testing.T) {
	rng := bbDnRng(t)
	for iter := 0; iter < 40; iter++ {
		a := bbDnRandDN(rng)
		// Shuffle RDN order; multiset must still match.
		rdns := append([]*ldap.RelativeDN(nil), a.RDNs...)
		rng.Shuffle(len(rdns), func(i, j int) { rdns[i], rdns[j] = rdns[j], rdns[i] })
		b := &ldap.DN{RDNs: rdns}
		if !a.RDNsMatch(b) {
			t.Fatalf("iter %d: shuffled RDNs should match", iter)
		}
		if !b.RDNsMatch(a) {
			t.Fatalf("iter %d: shuffled RDNs should match (reverse)", iter)
		}
	}
	// Duplicate-heavy cases.
	dup, _ := ldap.ParseDN("cn=x,cn=x,cn=y")
	twoOfEach, _ := ldap.ParseDN("cn=x,cn=y,cn=y")
	if dup.RDNsMatch(twoOfEach) || twoOfEach.RDNsMatch(dup) {
		t.Fatal("multiset: {x,x,y} must not match {x,y,y}")
	}
	sameDup, _ := ldap.ParseDN("cn=y,cn=x,cn=x")
	if !dup.RDNsMatch(sameDup) {
		t.Fatal("multiset: {x,x,y} should match {y,x,x}")
	}
	// Count mismatch never matches.
	one, _ := ldap.ParseDN("cn=x")
	if dup.RDNsMatch(one) || one.RDNsMatch(dup) {
		t.Fatal("count mismatch must not match")
	}
}

// Detail 9: AncestorOf = strictly shorter + equal trailing sequence;
// equal length -> false.
func TestDetail09_AncestorStrictness(t *testing.T) {
	rng := bbDnRng(t)
	for iter := 0; iter < 40; iter++ {
		d := bbDnRandDN(rng)
		// Prepend random RDNs -> d is ancestor.
		extra := bbDnRandDN(rng)
		child := &ldap.DN{RDNs: append(append([]*ldap.RelativeDN(nil), extra.RDNs...), d.RDNs...)}
		if !d.AncestorOf(child) {
			t.Fatalf("iter %d: d should be ancestor of prepended child", iter)
		}
		if child.AncestorOf(d) {
			t.Fatalf("iter %d: child must not be ancestor of shorter d", iter)
		}
		if d.AncestorOf(d) {
			t.Fatalf("iter %d: equal length must not be ancestor", iter)
		}
	}
	// Trailing-sequence mismatch.
	a, _ := ldap.ParseDN("ou=b,cn=z")
	b, _ := ldap.ParseDN("cn=x,ou=b,cn=y")
	if a.AncestorOf(b) || b.AncestorOf(a) {
		t.Fatal("trailing-sequence mismatch should not be ancestor")
	}
	// Empty DN is ancestor of any non-empty DN (vacuous suffix).
	empty, _ := ldap.ParseDN("")
	nonempty, _ := ldap.ParseDN("cn=a")
	if !empty.AncestorOf(nonempty) {
		t.Fatal("empty DN should be ancestor of non-empty DN")
	}
	if nonempty.AncestorOf(empty) {
		t.Fatal("non-empty cannot be ancestor of empty")
	}
}

// Detail 10: cert-subject conversion reverses attribute order, maps OIDs
// via the fixed table, errors on unknown OID / non-string value; the raw
// DER variant preserves multi-value RDNs (reversed internally too) and
// skips empty ones.
func TestDetail10_CertSubjectConversion(t *testing.T) {
	rng := bbDnRng(t)
	// FromCertSubject: build pkix.Name.Names with known OIDs; expect the DN
	// to hold them in reverse order.
	for iter := 0; iter < 40; iter++ {
		n := 1 + rng.Intn(5)
		types := make([]string, n)
		vals := make([]string, n)
		var names []pkix.AttributeTypeAndValue
		for i := 0; i < n; i++ {
			types[i] = bbDnTypes[rng.Intn(len(bbDnTypes))]
			vals[i] = bbDnRandValue(rng)
			names = append(names, pkix.AttributeTypeAndValue{
				Type:  asn1.ObjectIdentifier(bbDnOIDs[types[i]]),
				Value: vals[i],
			})
		}
		dn, err := ldap.FromCertSubject(pkix.Name{Names: names})
		if err != nil {
			t.Fatalf("iter %d: FromCertSubject err=%v", iter, err)
		}
		if len(dn.RDNs) != n {
			t.Fatalf("iter %d: %d RDNs want %d", iter, len(dn.RDNs), n)
		}
		for i := 0; i < n; i++ {
			// Reversed order.
			a := dn.RDNs[i].Attributes[0]
			if a.Type != types[n-1-i] || a.Value != vals[n-1-i] {
				t.Fatalf("iter %d rdn %d: got %s=%s want %s=%s", iter, i, a.Type, a.Value, types[n-1-i], vals[n-1-i])
			}
		}
	}
	// Unknown OID -> error.
	if _, err := ldap.FromCertSubject(pkix.Name{Names: []pkix.AttributeTypeAndValue{
		{Type: asn1.ObjectIdentifier{1, 2, 3, 4}, Value: "x"},
	}}); err == nil {
		t.Fatal("unknown OID should error")
	}
	// Non-string value -> error.
	if _, err := ldap.FromCertSubject(pkix.Name{Names: []pkix.AttributeTypeAndValue{
		{Type: asn1.ObjectIdentifier(bbDnOIDs["CN"]), Value: 42},
	}}); err == nil {
		t.Fatal("non-string value should error")
	}

	// FromRawCertSubject: DER via asn1.Marshal over pkix types.
	mkRaw := func(rdns []pkix.RelativeDistinguishedNameSET) []byte {
		raw, err := asn1.Marshal(pkix.RDNSequence(rdns))
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	atv := func(oid []int, val string) pkix.AttributeTypeAndValue {
		return pkix.AttributeTypeAndValue{Type: asn1.ObjectIdentifier(oid), Value: val}
	}

	for iter := 0; iter < 30; iter++ {
		// 1-3 RDNs, each with 1-2 attributes.
		nR := 1 + rng.Intn(3)
		var sets []pkix.RelativeDistinguishedNameSET
		var wantTypes [][]string
		var wantVals [][]string
		for i := 0; i < nR; i++ {
			nA := 1 + rng.Intn(2)
			var atvs []pkix.AttributeTypeAndValue
			var ts, vs []string
			for j := 0; j < nA; j++ {
				typ := bbDnTypes[rng.Intn(len(bbDnTypes))]
				val := bbDnRandValue(rng)
				atvs = append(atvs, atv(bbDnOIDs[typ], val))
				ts = append(ts, typ)
				vs = append(vs, val)
			}
			sets = append(sets, atvs)
			wantTypes = append(wantTypes, ts)
			wantVals = append(wantVals, vs)
		}
		raw := mkRaw(sets)
		dn, err := ldap.FromRawCertSubject(raw)
		if err != nil {
			t.Fatalf("iter %d: FromRawCertSubject err=%v", iter, err)
		}
		if len(dn.RDNs) != nR {
			t.Fatalf("iter %d: %d RDNs want %d", iter, len(dn.RDNs), nR)
		}
		for i := 0; i < nR; i++ {
			// RDN order is reversed; attributes within a RDN compare as a
			// multiset because DER SET OF encodes members canonically.
			src := nR - 1 - i
			want := map[string]int{}
			for j := range wantTypes[src] {
				want[wantTypes[src][j]+"\x00"+wantVals[src][j]]++
			}
			if len(dn.RDNs[i].Attributes) != len(wantTypes[src]) {
				t.Fatalf("iter %d rdn %d: %d attrs want %d", iter, i, len(dn.RDNs[i].Attributes), len(wantTypes[src]))
			}
			for _, a := range dn.RDNs[i].Attributes {
				k := a.Type + "\x00" + a.Value
				if want[k] == 0 {
					t.Fatalf("iter %d rdn %d: unexpected %s=%s", iter, i, a.Type, a.Value)
				}
				want[k]--
			}
			for k, c := range want {
				if c != 0 {
					t.Fatalf("iter %d rdn %d: missing %q", iter, i, k)
				}
			}
		}
	}
	// Empty SETs skipped.
	raw := mkRaw([]pkix.RelativeDistinguishedNameSET{
		{atv(bbDnOIDs["CN"], "x")},
		{},
	})
	dn, err := ldap.FromRawCertSubject(raw)
	if err != nil || len(dn.RDNs) != 1 {
		t.Fatalf("empty set skip: %+v err=%v", dn, err)
	}
	// Garbage DER -> error.
	if _, err := ldap.FromRawCertSubject([]byte{0x30, 0x03, 0x02, 0x01, 0x05}); err == nil {
		t.Fatal("non-Name DER should error")
	}
}
