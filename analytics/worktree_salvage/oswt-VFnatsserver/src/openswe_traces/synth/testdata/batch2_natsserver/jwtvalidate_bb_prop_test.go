// Hidden black-box property suite for the jwtvalidate unit.
// Drives only the API in api.md: ReadOperatorJWT/readOperatorJWT, wipeSlice,
// validateTrustedOperators, validateSrc, validateTimes/At,
// validateTimeRangeAt — fed with locally generated nkeys/jwt claims.
// One property per DETAILS.md line; seeded random inputs, no worked examples.
package server

import (
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

const bbJwtHiddenSeed = 20260919

func bbJwtSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return bbJwtHiddenSeed
}

func bbJwtRng(t *testing.T) *rand.Rand {
	t.Helper()
	return rand.New(rand.NewSource(bbJwtSeed()))
}

func bbJwtOpPub(t *testing.T) string {
	t.Helper()
	kp, err := nkeys.CreateOperator()
	if err != nil {
		t.Fatal(err)
	}
	pub, err := kp.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	return pub
}

func bbJwtAccPub(t *testing.T) string {
	t.Helper()
	kp, err := nkeys.CreateAccount()
	if err != nil {
		t.Fatal(err)
	}
	pub, err := kp.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	return pub
}

func bbJwtUserPub(t *testing.T) string {
	t.Helper()
	kp, err := nkeys.CreateUser()
	if err != nil {
		t.Fatal(err)
	}
	pub, err := kp.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	return pub
}

// bbJwtOperatorJWT returns a signed operator JWT string.
func bbJwtOperatorJWT(t *testing.T, mutate func(*jwt.OperatorClaims)) string {
	t.Helper()
	okp, err := nkeys.CreateOperator()
	if err != nil {
		t.Fatal(err)
	}
	opub, err := okp.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	oc := jwt.NewOperatorClaims(opub)
	if mutate != nil {
		mutate(oc)
	}
	tok, err := oc.Encode(okp)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// bbJwtUserJWT returns a signed user JWT string.
func bbJwtUserJWT(t *testing.T, mutate func(*jwt.UserClaims)) string {
	t.Helper()
	akp, err := nkeys.CreateAccount()
	if err != nil {
		t.Fatal(err)
	}
	uc := jwt.NewUserClaims(bbJwtUserPub(t))
	if mutate != nil {
		mutate(uc)
	}
	tok, err := uc.Encode(akp)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// Detail 1: loader — on file-read failure, the argument itself is used as
// JWT text only when it starts with "eyJ"; otherwise the read error is
// returned.
func TestDetail01_LoaderFallback(t *testing.T) {
	tok := bbJwtOperatorJWT(t, nil)
	if !strings.HasPrefix(tok, jwtPrefix) {
		t.Fatalf("generated JWT lacks %q prefix", jwtPrefix)
	}
	// Nonexistent path, but the arg IS the JWT (starts with eyJ) -> parsed.
	oc, err := ReadOperatorJWT(tok)
	if err != nil || oc == nil {
		t.Fatalf("JWT-as-arg: oc=%v err=%v", oc, err)
	}
	// Nonexistent path not starting with eyJ -> read error.
	_, err = ReadOperatorJWT(filepath.Join(t.TempDir(), "no-such.jwt"))
	if err == nil {
		t.Fatal("missing file should error")
	}
	// Path-like garbage that is not a JWT prefix -> read error, not parse.
	_, err = ReadOperatorJWT("/definitely/missing/file.jwt")
	if err == nil {
		t.Fatal("missing path should error")
	}
	// An existing file that is not a JWT -> parse error (file read wins).
	bad := filepath.Join(t.TempDir(), "bad.jwt")
	if err := os.WriteFile(bad, []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadOperatorJWT(bad); err == nil {
		t.Fatal("non-JWT file should error")
	}
	// A real file containing a JWT parses.
	good := filepath.Join(t.TempDir(), "op.jwt")
	if err := os.WriteFile(good, []byte(tok), 0o644); err != nil {
		t.Fatal(err)
	}
	oc, err = ReadOperatorJWT(good)
	if err != nil || oc == nil {
		t.Fatalf("file JWT: oc=%v err=%v", oc, err)
	}
}

// Detail 2: loaded content is parsed decorated-or-bare into operator
// claims; the buffer is 'x'-wiped before returning; the exported form
// hides the raw JWT.
func TestDetail02_DecoratedOrBare(t *testing.T) {
	tok := bbJwtOperatorJWT(t, nil)
	// Bare JWT.
	raw, oc, err := readOperatorJWT(tok)
	if err != nil || oc == nil {
		t.Fatalf("bare: err=%v", err)
	}
	if raw == "" {
		t.Fatal("internal form should return raw JWT")
	}
	// Decorated: -----BEGIN NATS OPERATOR JWT----- wrapper.
	decorated := "-----BEGIN NATS OPERATOR JWT-----\n" + tok + "\n------END NATS OPERATOR JWT-----\n"
	fp := filepath.Join(t.TempDir(), "op.jwt")
	if err := os.WriteFile(fp, []byte(decorated), 0o644); err != nil {
		t.Fatal(err)
	}
	raw2, oc2, err := readOperatorJWT(fp)
	if err != nil || oc2 == nil {
		t.Fatalf("decorated: err=%v", err)
	}
	if raw2 == "" {
		t.Fatal("decorated raw should be non-empty")
	}
	// Exported form hides the raw JWT — signature only returns claims.
	oc3, err := ReadOperatorJWT(tok)
	if err != nil || oc3 == nil {
		t.Fatalf("exported: %v", err)
	}
	if oc3.Subject != oc.Subject {
		t.Fatalf("subjects differ: %q vs %q", oc3.Subject, oc.Subject)
	}
	// wipeSlice fills with 'x'.
	buf := []byte("secret-jwt-bytes")
	wipeSlice(buf)
	for i, b := range buf {
		if b != 'x' {
			t.Fatalf("buf[%d]=%q want 'x'", i, b)
		}
	}
	// Empty wipe is safe.
	wipeSlice(nil)
}

// Detail 3: no operators -> default sentinel forbidden (error); any other
// no-operator options fine.
func TestDetail03_NoOperatorsSentinel(t *testing.T) {
	sentinel := bbJwtUserJWT(t, func(uc *jwt.UserClaims) { uc.BearerToken = true })
	o := &Options{DefaultSentinel: sentinel}
	if err := validateTrustedOperators(o); err == nil {
		t.Fatal("default sentinel with no operators should error")
	}
	for _, o := range []*Options{
		{},
		{Users: []*User{{Username: "u", Password: "p"}}},
		{Nkeys: []*NkeyUser{{Nkey: "UPUB"}}},
		{Accounts: []*Account{NewAccount("A")}},
		{Authorization: "tok"},
	} {
		if err := validateTrustedOperators(o); err != nil {
			t.Fatalf("no-operator options %+v: %v", o, err)
		}
	}
}

// Detail 4: sentinel decode + admission — must decode as user claims;
// non-bearer accepted only with issuer-account + empty permissions;
// other non-bearer -> "must be a bearer token".
func TestDetail04_SentinelAdmission(t *testing.T) {
	sysAcc := bbJwtAccPub(t)
	mkopts := func(sentinel string) *Options {
		op := bbJwtOperatorJWT(t, func(oc *jwt.OperatorClaims) { oc.SystemAccount = sysAcc })
		raw, opc, err := readOperatorJWT(op)
		if err != nil || opc == nil {
			t.Fatalf("operator claims: %v", err)
		}
		_ = raw
		return &Options{
			TrustedOperators: []*jwt.OperatorClaims{opc},
			AccountResolver:  &MemAccResolver{},
			SystemAccount:    sysAcc,
			DefaultSentinel:  sentinel,
		}
	}
	// Bearer token sentinel -> admitted.
	bearer := bbJwtUserJWT(t, func(uc *jwt.UserClaims) { uc.BearerToken = true })
	if err := validateTrustedOperators(mkopts(bearer)); err != nil {
		t.Fatalf("bearer sentinel: %v", err)
	}
	// Non-bearer + issuer account + empty permissions -> admitted (scoped).
	// NewUserClaims pre-fills non-empty limits; SetScoped(true) empties them.
	scoped := bbJwtUserJWT(t, func(uc *jwt.UserClaims) {
		uc.BearerToken = false
		uc.SetScoped(true)
		uc.IssuerAccount = bbJwtAccPub(t)
	})
	if err := validateTrustedOperators(mkopts(scoped)); err != nil {
		t.Fatalf("scoped non-bearer sentinel: %v", err)
	}
	// Non-bearer without issuer account -> "must be a bearer token".
	plain := bbJwtUserJWT(t, func(uc *jwt.UserClaims) { uc.BearerToken = false })
	err := validateTrustedOperators(mkopts(plain))
	if err == nil {
		t.Fatal("non-scoped non-bearer sentinel should error")
	}
	if !strings.Contains(err.Error(), "bearer") {
		t.Fatalf("sentinel err=%q want mention of bearer", err)
	}
	// Non-bearer with issuer account but NON-empty permissions -> error.
	perm := bbJwtUserJWT(t, func(uc *jwt.UserClaims) {
		uc.BearerToken = false
		uc.SetScoped(true)
		uc.IssuerAccount = bbJwtAccPub(t)
		uc.Permissions.Pub.Allow = jwt.StringList{"foo"}
	})
	if err := validateTrustedOperators(mkopts(perm)); err == nil {
		t.Fatal("non-bearer with permissions should error")
	}
	// Undecodable sentinel -> error.
	if err := validateTrustedOperators(mkopts("not-a-jwt")); err == nil {
		t.Fatal("undecodable sentinel should error")
	}
}

// Detail 5: operator-mode exclusions — account resolver required;
// Accounts/Users/Nkeys/AuthCallout rejected; TrustedKeys+TrustedOperators
// conflict.
func TestDetail05_OperatorModeExclusions(t *testing.T) {
	sysAcc := bbJwtAccPub(t)
	op := bbJwtOperatorJWT(t, func(oc *jwt.OperatorClaims) { oc.SystemAccount = sysAcc })
	_, opc, err := readOperatorJWT(op)
	if err != nil || opc == nil {
		t.Fatalf("operator claims: %v", err)
	}
	base := func() *Options {
		return &Options{
			TrustedOperators: []*jwt.OperatorClaims{opc},
			AccountResolver:  &MemAccResolver{},
			SystemAccount:    sysAcc,
		}
	}
	if err := validateTrustedOperators(base()); err != nil {
		t.Fatalf("baseline operator config: %v", err)
	}
	// Resolver required.
	o := base()
	o.AccountResolver = nil
	if err := validateTrustedOperators(o); err == nil {
		t.Fatal("operators without resolver should error")
	}
	// Accounts/Users/Nkeys/AuthCallout rejected in operator mode.
	o = base()
	o.Accounts = []*Account{NewAccount("A")}
	if err := validateTrustedOperators(o); err == nil {
		t.Fatal("Accounts in operator mode should error")
	}
	o = base()
	o.Users = []*User{{Username: "u", Password: "p"}}
	if err := validateTrustedOperators(o); err == nil {
		t.Fatal("Users in operator mode should error")
	}
	o = base()
	o.Nkeys = []*NkeyUser{{Nkey: "UPUB"}}
	if err := validateTrustedOperators(o); err == nil {
		t.Fatal("Nkeys in operator mode should error")
	}
	o = base()
	o.AuthCallout = &AuthCallout{}
	if err := validateTrustedOperators(o); err == nil {
		t.Fatal("AuthCallout in operator mode should error")
	}
	// TrustedKeys + TrustedOperators conflict.
	o = base()
	o.TrustedKeys = []string{bbJwtOpPub(t)}
	if err := validateTrustedOperators(o); err == nil {
		t.Fatal("TrustedKeys+TrustedOperators should error")
	}
}

// Detail 6: system-account matching — config value must appear in some
// operator's claims when any operator declares one; unset-both + dir-type
// resolver -> error.
func TestDetail06_SystemAccountMatching(t *testing.T) {
	sysA, sysB := bbJwtAccPub(t), bbJwtAccPub(t)
	op := bbJwtOperatorJWT(t, func(oc *jwt.OperatorClaims) { oc.SystemAccount = sysA })
	_, opc, err := readOperatorJWT(op)
	if err != nil {
		t.Fatal(err)
	}
	// Match -> ok.
	o := &Options{TrustedOperators: []*jwt.OperatorClaims{opc},
		AccountResolver: &MemAccResolver{}, SystemAccount: sysA}
	if err := validateTrustedOperators(o); err != nil {
		t.Fatalf("matching sys account: %v", err)
	}
	// Mismatch -> error.
	o.SystemAccount = sysB
	if err := validateTrustedOperators(o); err == nil {
		t.Fatal("mismatched system account should error")
	}
	// Operator declares a sys account but config unset -> error.
	o.SystemAccount = ""
	if err := validateTrustedOperators(o); err == nil {
		t.Fatal("unset system account (operator declares) should error")
	}
	// No operator declares one + unset config + dir resolver -> error.
	op2 := bbJwtOperatorJWT(t, func(oc *jwt.OperatorClaims) { oc.SystemAccount = "" })
	_, opc2, err := readOperatorJWT(op2)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	dr, err := NewDirAccResolver(dir, 1<<20, time.Minute, deleteType(0))
	if err != nil {
		t.Fatal(err)
	}
	o = &Options{TrustedOperators: []*jwt.OperatorClaims{opc2},
		AccountResolver: dr, SystemAccount: ""}
	if err := validateTrustedOperators(o); err == nil {
		t.Fatal("unset-both + dir resolver should error")
	}
	// Same with a memory resolver -> ok.
	o.AccountResolver = &MemAccResolver{}
	if err := validateTrustedOperators(o); err != nil {
		t.Fatalf("unset-both + mem resolver: %v", err)
	}
}

// Detail 7: per-operator minimum-version check — parse error -> error;
// strictly-greater major/minor/update -> error; equal or lower -> ok.
func TestDetail07_AssertServerVersion(t *testing.T) {
	sysAcc := bbJwtAccPub(t)
	mk := func(assert string) *Options {
		op := bbJwtOperatorJWT(t, func(oc *jwt.OperatorClaims) {
			oc.SystemAccount = sysAcc
			oc.AssertServerVersion = assert
		})
		_, opc, err := readOperatorJWT(op)
		if err != nil {
			t.Fatal(err)
		}
		return &Options{TrustedOperators: []*jwt.OperatorClaims{opc},
			AccountResolver: &MemAccResolver{}, SystemAccount: sysAcc}
	}
	// Below current version -> ok.
	if err := validateTrustedOperators(mk("1.0.0")); err != nil {
		t.Fatalf("low assert: %v", err)
	}
	// Absurdly higher -> error.
	if err := validateTrustedOperators(mk("99.0.0")); err == nil {
		t.Fatal("high assert should error")
	}
	// Just above current major/minor/patch -> error (VERSION is 2.x here).
	if err := validateTrustedOperators(mk("3.0.0")); err == nil {
		t.Fatal("3.0.0 assert should error")
	}
	// Parse failure -> error.
	for _, bad := range []string{"abc", "1.2", "1.2.3.4", "x.y.z"} {
		if err := validateTrustedOperators(mk(bad)); err == nil {
			t.Fatalf("unparseable assert %q should error", bad)
		}
	}
}

// Detail 8: TrustedKeys expansion — operator subject skipped iff
// StrictSigningKeyUsage; all SigningKeys always appended; every key
// validated as public operator nkey.
func TestDetail08_TrustedKeysExpansion(t *testing.T) {
	sysAcc := bbJwtAccPub(t)
	strictSubj := bbJwtOpPub(t)
	nonStrictSubj := bbJwtOpPub(t)
	sk1, sk2, sk3 := bbJwtOpPub(t), bbJwtOpPub(t), bbJwtOpPub(t)
	mk := func(strict bool, subj string, keys ...string) *jwt.OperatorClaims {
		op := bbJwtOperatorJWT(t, func(oc *jwt.OperatorClaims) {
			oc.Subject = subj
			oc.SystemAccount = sysAcc
			oc.StrictSigningKeyUsage = strict
			oc.SigningKeys = jwt.StringList(keys)
		})
		_, opc, err := readOperatorJWT(op)
		if err != nil {
			t.Fatal(err)
		}
		return opc
	}
	opcStrict := mk(true, strictSubj, sk1, sk2)
	opcOpen := mk(false, nonStrictSubj, sk3)
	// TrustedKeys and TrustedOperators are mutually exclusive inputs, so
	// expansion must start from an empty TrustedKeys.
	o := &Options{
		TrustedOperators: []*jwt.OperatorClaims{opcStrict, opcOpen},
		AccountResolver:  &MemAccResolver{},
		SystemAccount:    sysAcc,
	}
	if err := validateTrustedOperators(o); err != nil {
		t.Fatalf("expansion: %v", err)
	}
	got := map[string]bool{}
	for _, k := range o.TrustedKeys {
		got[k] = true
	}
	for _, want := range []string{sk1, sk2, sk3, nonStrictSubj} {
		if !got[want] {
			t.Fatalf("TrustedKeys missing %q: %v", want, o.TrustedKeys)
		}
	}
	if got[strictSubj] {
		t.Fatalf("strict operator subject %q should be skipped", strictSubj)
	}
	// Every key validated as public operator nkey.
	bad := &Options{
		TrustedOperators: []*jwt.OperatorClaims{opcStrict},
		AccountResolver:  &MemAccResolver{},
		SystemAccount:    sysAcc,
		TrustedKeys:      []string{"NOT-A-KEY"},
	}
	if err := validateTrustedOperators(bad); err == nil {
		t.Fatal("invalid trusted key should error")
	}
	// Account key (A-prefix) is not an operator nkey.
	bad.TrustedKeys = []string{bbJwtAccPub(t)}
	if err := validateTrustedOperators(bad); err == nil {
		t.Fatal("account-prefixed trusted key should error")
	}
}

// Detail 9: pinned accounts validated as account nkeys; configured system
// account auto-pinned — but ONLY when the pinned set is already non-empty.
func TestDetail09_PinnedAccounts(t *testing.T) {
	sysAcc := bbJwtAccPub(t)
	op := bbJwtOperatorJWT(t, func(oc *jwt.OperatorClaims) { oc.SystemAccount = sysAcc })
	_, opc, err := readOperatorJWT(op)
	if err != nil {
		t.Fatal(err)
	}
	pin := bbJwtAccPub(t)
	mk := func(pinned map[string]struct{}) *Options {
		return &Options{
			TrustedOperators:       []*jwt.OperatorClaims{opc},
			AccountResolver:        &MemAccResolver{},
			SystemAccount:          sysAcc,
			resolverPinnedAccounts: pinned,
		}
	}
	// Valid pin -> ok; system account auto-pinned into the non-empty set.
	o := mk(map[string]struct{}{pin: {}})
	if err := validateTrustedOperators(o); err != nil {
		t.Fatalf("valid pin: %v", err)
	}
	if _, ok := o.resolverPinnedAccounts[pin]; !ok {
		t.Fatal("pin dropped")
	}
	if _, ok := o.resolverPinnedAccounts[sysAcc]; !ok {
		t.Fatal("system account not auto-pinned into non-empty set")
	}
	// Invalid pin -> error.
	o = mk(map[string]struct{}{"NOT-AN-ACCOUNT": {}})
	if err := validateTrustedOperators(o); err == nil {
		t.Fatal("invalid pinned account should error")
	}
	// Operator key (O-prefix) is not an account nkey.
	o = mk(map[string]struct{}{bbJwtOpPub(t): {}})
	if err := validateTrustedOperators(o); err == nil {
		t.Fatal("operator-key pin should error")
	}
	// Empty/nil pinned set skips the block — no auto-pin, no error.
	o = mk(nil)
	if err := validateTrustedOperators(o); err != nil {
		t.Fatalf("nil pinned set: %v", err)
	}
	if len(o.resolverPinnedAccounts) != 0 {
		t.Fatalf("nil set grew: %v", o.resolverPinnedAccounts)
	}
	o = mk(map[string]struct{}{})
	if err := validateTrustedOperators(o); err != nil {
		t.Fatalf("empty pinned set: %v", err)
	}
	if len(o.resolverPinnedAccounts) != 0 {
		t.Fatalf("empty set grew: %v", o.resolverPinnedAccounts)
	}
}

// Detail 10: validateSrc — nil claims->false; empty Src->true; empty
// host->false; unparseable host->false; unparseable CIDR->false; any
// containing CIDR->true.
func TestDetail10_ValidateSrc(t *testing.T) {
	if validateSrc(nil, "1.2.3.4") {
		t.Fatal("nil claims should be false")
	}
	uc := &jwt.UserClaims{}
	if !validateSrc(uc, "1.2.3.4") {
		t.Fatal("empty Src should admit")
	}
	if !validateSrc(uc, "999.999.999.999") {
		t.Fatal("empty Src should admit even odd hosts")
	}
	uc.Src = jwt.CIDRList{"10.0.0.0/8"}
	if validateSrc(uc, "") {
		t.Fatal("empty host should be false")
	}
	if validateSrc(uc, "not-an-ip") {
		t.Fatal("unparseable host should be false")
	}
	if !validateSrc(uc, "10.20.30.40") {
		t.Fatal("in-CIDR host should admit")
	}
	if validateSrc(uc, "11.0.0.1") {
		t.Fatal("out-of-CIDR host should be false")
	}
	// Multiple CIDRs: any containing admits.
	uc.Src = jwt.CIDRList{"192.168.0.0/16", "10.0.0.0/8"}
	if !validateSrc(uc, "192.168.1.1") || !validateSrc(uc, "10.9.9.9") {
		t.Fatal("any containing CIDR should admit")
	}
	if validateSrc(uc, "172.16.0.1") {
		t.Fatal("outside all CIDRs should be false")
	}
	// Unparseable CIDR -> false.
	uc.Src = jwt.CIDRList{"not-a-cidr"}
	if validateSrc(uc, "10.0.0.1") {
		t.Fatal("unparseable CIDR should be false")
	}
	// v6 CIDR.
	uc.Src = jwt.CIDRList{"2001:db8::/32"}
	if !validateSrc(uc, "2001:db8::1") {
		t.Fatal("v6 CIDR should admit")
	}
	if validateSrc(uc, "2001:db9::1") {
		t.Fatal("v6 out-of-CIDR should be false")
	}
}

// Detail 11: validateTimesAt — nil->(false,0); empty Times->(true,0);
// locale load failure->(false,0); now converted into locale; start/end
// parsed "15:04:05" in locale and anchored to now's date.
func TestDetail11_ValidateTimesAtBasics(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	ok, d := validateTimesAt(nil, now)
	if ok || d != 0 {
		t.Fatalf("nil claims=(%v,%v) want (false,0)", ok, d)
	}
	ok, d = validateTimesAt(&jwt.UserClaims{}, now)
	if !ok || d != 0 {
		t.Fatalf("empty times=(%v,%v) want (true,0)", ok, d)
	}
	// Locale load failure -> (false,0).
	uc := &jwt.UserClaims{}
	uc.Locale = "Bogus/Zone"
	uc.Times = []jwt.TimeRange{{Start: "00:00:00", End: "23:59:59"}}
	ok, d = validateTimesAt(uc, now)
	if ok || d != 0 {
		t.Fatalf("bad locale=(%v,%v) want (false,0)", ok, d)
	}
	// Locale conversion: now=15:30 UTC is 10:30 in America/New_York (EST).
	uc = &jwt.UserClaims{}
	uc.Locale = "America/New_York"
	uc.Times = []jwt.TimeRange{{Start: "10:00:00", End: "11:00:00"}}
	nowUTC := time.Date(2026, 1, 15, 15, 30, 0, 0, time.UTC)
	ok, d = validateTimesAt(uc, nowUTC)
	if !ok {
		t.Fatal("NY window should admit 10:30 local")
	}
	if d != 30*time.Minute {
		t.Fatalf("remaining=%v want 30m", d)
	}
	// Same wall clock but window in UTC misses.
	uc.Locale = "UTC"
	ok, _ = validateTimesAt(uc, nowUTC)
	if ok {
		t.Fatal("UTC 10:00-11:00 window should not admit 15:30 UTC")
	}
	// Anchoring: window bounds use NOW's date, not arbitrary dates.
	uc.Times = []jwt.TimeRange{{Start: "15:00:00", End: "16:00:00"}}
	ok, d = validateTimesAt(uc, nowUTC)
	if !ok || d != 30*time.Minute {
		t.Fatalf("anchored window=(%v,%v) want (true,30m)", ok, d)
	}
	// validateTimes (no-at variant) agrees on trivial cases.
	ok, d = validateTimes(nil)
	if ok || d != 0 {
		t.Fatalf("validateTimes(nil)=(%v,%v)", ok, d)
	}
	ok, d = validateTimes(&jwt.UserClaims{})
	if !ok || d != 0 {
		t.Fatalf("validateTimes(empty)=(%v,%v)", ok, d)
	}
}

// Detail 12: window boundaries are STRICT — now==start or now==end does
// not admit.
func TestDetail12_StrictBoundaries(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	uc := &jwt.UserClaims{}
	uc.Locale = "UTC"
	uc.Times = []jwt.TimeRange{{Start: "12:00:00", End: "13:00:00"}}
	ok, _ := validateTimesAt(uc, now)
	if ok {
		t.Fatal("now==start should not admit (strict)")
	}
	uc.Times = []jwt.TimeRange{{Start: "11:00:00", End: "12:00:00"}}
	ok, _ = validateTimesAt(uc, now)
	if ok {
		t.Fatal("now==end should not admit (strict)")
	}
	// Just inside admits.
	uc.Times = []jwt.TimeRange{{Start: "11:59:59", End: "12:00:01"}}
	ok, d := validateTimesAt(uc, now)
	if !ok || d != time.Second {
		t.Fatalf("inside window=(%v,%v) want (true,1s)", ok, d)
	}
	// validateTimeRangeAt agrees: anchored times directly.
	start := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 15, 13, 0, 0, 0, time.UTC)
	ok, _ = validateTimeRangeAt(start, end, now)
	if ok {
		t.Fatal("rangeAt now==start should not admit")
	}
	ok, _ = validateTimeRangeAt(start, end, end)
	if ok {
		t.Fatal("rangeAt now==end should not admit")
	}
	ok, d = validateTimeRangeAt(start, end, now.Add(time.Second))
	if !ok || d != time.Hour-time.Second {
		t.Fatalf("rangeAt inside=(%v,%v)", ok, d)
	}
}

// Detail 13: overnight windows (start>end): now<end admits end−now;
// now>start admits end+1day−now; equal start/end admits nothing.
func TestDetail13_OvernightWindows(t *testing.T) {
	uc := &jwt.UserClaims{}
	uc.Locale = "UTC"
	uc.Times = []jwt.TimeRange{{Start: "22:00:00", End: "02:00:00"}}
	// Before midnight: now=23:00 -> end+1day-now = 3h.
	now := time.Date(2026, 1, 15, 23, 0, 0, 0, time.UTC)
	ok, d := validateTimesAt(uc, now)
	if !ok || d != 3*time.Hour {
		t.Fatalf("pre-midnight=(%v,%v) want (true,3h)", ok, d)
	}
	// After midnight: now=01:00 -> end-now = 1h.
	now = time.Date(2026, 1, 16, 1, 0, 0, 0, time.UTC)
	ok, d = validateTimesAt(uc, now)
	if !ok || d != time.Hour {
		t.Fatalf("post-midnight=(%v,%v) want (true,1h)", ok, d)
	}
	// Outside: now=03:00 -> false.
	now = time.Date(2026, 1, 16, 3, 0, 0, 0, time.UTC)
	ok, _ = validateTimesAt(uc, now)
	if ok {
		t.Fatal("03:00 outside overnight window should be false")
	}
	// Outside on the day side too.
	now = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	ok, _ = validateTimesAt(uc, now)
	if ok {
		t.Fatal("noon outside overnight window should be false")
	}
	// Equal start/end admits nothing at any time.
	uc.Times = []jwt.TimeRange{{Start: "12:00:00", End: "12:00:00"}}
	for _, h := range []int{0, 6, 12, 18, 23} {
		now = time.Date(2026, 1, 15, h, 0, 0, 0, time.UTC)
		ok, _ = validateTimesAt(uc, now)
		if ok {
			t.Fatalf("degenerate window admitted at %02d:00", h)
		}
	}
}

// Detail 14: multiple windows — result true iff any admits; duration = MAX
// remaining across admitting windows.
func TestDetail14_MultipleWindows(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	uc := &jwt.UserClaims{}
	uc.Locale = "UTC"
	uc.Times = []jwt.TimeRange{
		{Start: "08:00:00", End: "10:00:00"}, // missed
		{Start: "11:00:00", End: "13:00:00"}, // admits, 1h left
		{Start: "11:30:00", End: "18:00:00"}, // admits, 6h left (max)
		{Start: "20:00:00", End: "22:00:00"}, // future
	}
	ok, d := validateTimesAt(uc, now)
	if !ok {
		t.Fatal("multi-window should admit")
	}
	if d != 6*time.Hour {
		t.Fatalf("remaining=%v want 6h (max across admitting)", d)
	}
	// None admitting -> false.
	uc.Times = []jwt.TimeRange{
		{Start: "01:00:00", End: "02:00:00"},
		{Start: "20:00:00", End: "22:00:00"},
	}
	ok, _ = validateTimesAt(uc, now)
	if ok {
		t.Fatal("no admitting window should be false")
	}
	// Randomized cross-check vs local oracle (UTC only).
	rng := bbJwtRng(t)
	for i := 0; i < 60; i++ {
		var trs []jwt.TimeRange
		n := 1 + rng.Intn(4)
		for j := 0; j < n; j++ {
			s := time.Duration(rng.Intn(86400)) * time.Second
			e := time.Duration(rng.Intn(86400)) * time.Second
			trs = append(trs, jwt.TimeRange{
				Start: fmt15(s), End: fmt15(e)})
		}
		uc.Times = trs
		secNow := rng.Intn(86400)
		now := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC).Add(time.Duration(secNow) * time.Second)
		gotOk, gotD := validateTimesAt(uc, now)
		wantOk, wantD := bbTimeOracle(trs, secNow)
		if gotOk != wantOk || (wantOk && gotD != wantD) {
			t.Fatalf("iter %d: times=%v now=%02d:%02d:%02d got=(%v,%v) want=(%v,%v)",
				i, trs, secNow/3600, (secNow/60)%60, secNow%60, gotOk, gotD, wantOk, wantD)
		}
	}
}

func fmt15(d time.Duration) string {
	d = d % (24 * time.Hour)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return time.Date(2000, 1, 1, h, m, s, 0, time.UTC).Format("15:04:05")
}

// bbTimeOracle computes admission + max-remaining per the documented rules:
// strict boundaries; overnight wraps; equal start/end admits nothing.
func bbTimeOracle(trs []jwt.TimeRange, nowSec int) (bool, time.Duration) {
	const day = 86400
	var best time.Duration
	any := false
	for _, tr := range trs {
		p, _ := time.Parse("15:04:05", tr.Start)
		s := p.Hour()*3600 + p.Minute()*60 + p.Second()
		p, _ = time.Parse("15:04:05", tr.End)
		e := p.Hour()*3600 + p.Minute()*60 + p.Second()
		var rem int
		var ok bool
		switch {
		case s == e:
			ok = false
		case s < e:
			ok = nowSec > s && nowSec < e
			rem = e - nowSec
		default: // overnight
			if nowSec < e {
				ok = true
				rem = e - nowSec
			} else if nowSec > s {
				ok = true
				rem = e + day - nowSec
			}
		}
		if ok {
			any = true
			if time.Duration(rem)*time.Second > best {
				best = time.Duration(rem) * time.Second
			}
		}
	}
	return any, best
}
