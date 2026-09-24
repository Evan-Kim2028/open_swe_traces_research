package server

import (
	"testing"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

func eaBbWant(t *testing.T, got, want bool, what string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: got %v, want %v", what, got, want)
	}
}

// TestDetail01: checkStreamImportAuthorizedNoLock returns false when the
// account has no stream exports or when subject fails IsValidSubject; the
// service variant omits the validity check.
func TestDetail01(t *testing.T) {
	exp, imp := NewAccount("EXP"), NewAccount("IMP")

	eaBbWant(t, exp.checkStreamImportAuthorized(imp, "foo", nil), false, "no stream exports")
	eaBbWant(t, exp.checkServiceImportAuthorized(imp, "foo", nil), false, "no service exports")

	exp.AddStreamExport(">", nil)
	eaBbWant(t, exp.checkStreamImportAuthorized(imp, "a..b", nil), false, "invalid stream subject must fail")
	eaBbWant(t, exp.checkStreamImportAuthorized(imp, "a b", nil), false, "invalid stream subject must fail")

	expS := NewAccount("EXPS")
	expS.AddServiceExport(">", nil)
	eaBbWant(t, expS.checkServiceImportAuthorized(imp, "a..b", nil), true,
		"service variant omits validity check; covered invalid subject authorized")
}

// TestDetail02: checkStreamExportApproved tries an exact map hit first;
// ea == nil means a public export and authorizes any account.
func TestDetail02(t *testing.T) {
	exp, imp, imp2 := NewAccount("EXP"), NewAccount("IMP"), NewAccount("IMP2")

	exp.AddStreamExport("foo", nil)
	eaBbWant(t, exp.checkStreamImportAuthorized(imp, "foo", nil), true, "public export authorizes")
	eaBbWant(t, exp.checkStreamImportAuthorized(imp2, "foo", nil), true, "public export authorizes any account")
	eaBbWant(t, exp.checkStreamImportAuthorized(imp, "bar", nil), false, "unrelated subject not authorized")
	eaBbWant(t, exp.checkStreamImportAuthorized(imp, "foo.bar", nil), false, "literal export does not cover longer subject")
}

// TestDetail03: on a map miss all export subjects are iterated with
// isSubsetMatch(subjectTokens, exportSubj); the requesting subject must be
// a subset of (covered by) the export pattern.
func TestDetail03(t *testing.T) {
	exp, imp := NewAccount("EXP"), NewAccount("IMP")

	exp.AddStreamExport("foo.*", nil)
	eaBbWant(t, exp.checkStreamImportAuthorized(imp, "foo.bar", nil), true, "subset of export pattern")
	eaBbWant(t, exp.checkStreamImportAuthorized(imp, "foo.*", nil), true, "identical pattern is a subset")
	eaBbWant(t, exp.checkStreamImportAuthorized(imp, "foo.bar.baz", nil), false, "request broader than export pattern")
	eaBbWant(t, exp.checkStreamImportAuthorized(imp, "foo.>", nil), false, "wildcard request broader than export")
	eaBbWant(t, exp.checkStreamImportAuthorized(imp, "bar.x", nil), false, "unmatched prefix")

	exp.AddStreamExport("deep.>", nil)
	eaBbWant(t, exp.checkStreamImportAuthorized(imp, "deep.a.b.c", nil), true, "full wildcard covers tail")
	eaBbWant(t, exp.checkStreamImportAuthorized(imp, "deep", nil), false, "fwc export requires the prefix")
}

// TestDetail04: checkAuth order — public, then accountPos (1-indexed position
// in the requesting subject), then tokenReq via checkActivation, then the
// approved list.
func TestDetail04(t *testing.T) {
	imp, other := NewAccount("IMP"), NewAccount("OTHER")

	// accountPos decides before the approved list and without a token.
	exp := NewAccount("EXP")
	if err := exp.addStreamExportWithAccountPos("foo.*", []*Account{other}, 2); err != nil {
		t.Fatalf("addStreamExportWithAccountPos: %v", err)
	}
	eaBbWant(t, exp.checkStreamImportAuthorized(imp, "foo.IMP", nil), true,
		"account name at position authorizes even when not on approved list")
	eaBbWant(t, exp.checkStreamImportAuthorized(other, "foo.X", nil), false,
		"position mismatch denies even an approved-list member")
	eaBbWant(t, exp.checkStreamImportAuthorized(other, "foo.OTHER", nil), true,
		"position match authorizes")

	// tokenReq requires an activation token.
	exp2 := NewAccount("EXP2")
	exp2.AddStreamExport("sec", []*Account{})
	eaBbWant(t, exp2.checkStreamImportAuthorized(imp, "sec", nil), false, "tokenReq denies claimless import")

	// approved list membership decides when no accountPos and no tokenReq.
	exp3 := NewAccount("EXP3")
	exp3.AddStreamExport("sec", []*Account{imp})
	eaBbWant(t, exp3.checkStreamImportAuthorized(imp, "sec", nil), true, "approved member authorized")
	eaBbWant(t, exp3.checkStreamImportAuthorized(other, "sec", nil), false, "non-member denied")

	// checkActivation: a valid activation token authorizes; a bad or
	// revoked token does not.
	expKP, _ := nkeys.CreateAccount()
	expPub, _ := expKP.PublicKey()
	impKP, _ := nkeys.CreateAccount()
	impPub, _ := impKP.PublicKey()
	expA, impA := NewAccount(expPub), NewAccount(impPub)
	expA.AddStreamExport("priv", []*Account{})

	act := jwt.NewActivationClaims(impPub)
	act.ImportSubject = "priv"
	act.ImportType = jwt.Stream
	tok, err := act.Encode(expKP)
	if err != nil {
		t.Fatalf("encode activation: %v", err)
	}
	claim := &jwt.Import{Account: expPub, Subject: "priv", Type: jwt.Stream, Token: tok}
	eaBbWant(t, expA.checkStreamImportAuthorized(impA, "priv", claim), true, "valid activation token authorizes")
	eaBbWant(t, expA.checkStreamImportAuthorized(impA, "priv",
		&jwt.Import{Account: expPub, Subject: "priv", Type: jwt.Stream, Token: "garbage"}), false,
		"undecodable token denied")

	expB := NewAccount(expPub)
	expB.AddStreamExport("priv", []*Account{})
	expB.exports.streams["priv"].actsRevoked = map[string]int64{impPub: act.IssuedAt + 100}
	eaBbWant(t, expB.checkStreamImportAuthorized(impA, "priv", claim), false, "revoked activation denied")

	// tokenReq applies to service exports too (same checkAuth predicate).
	expC := NewAccount("EXPC")
	expC.AddServiceExport("req.echo", []*Account{})
	eaBbWant(t, expC.checkServiceImportAuthorized(impA, "req.echo", nil), false, "service tokenReq denies claimless")

	sact := jwt.NewActivationClaims(impPub)
	sact.ImportSubject = "req.echo"
	sact.ImportType = jwt.Service
	stok, err := sact.Encode(expKP)
	if err != nil {
		t.Fatalf("encode service activation: %v", err)
	}
	expD := NewAccount(expPub)
	expD.AddServiceExport("req.echo", []*Account{})
	eaBbWant(t, expD.checkServiceImportAuthorized(impA, "req.echo",
		&jwt.Import{Account: expPub, Subject: "req.echo", Type: jwt.Service, Token: stok}), true,
		"valid service activation authorizes")
}

// TestDetail05: getServiceExport returns the exact map hit else falls back
// to iterating service exports with isSubsetMatch; getWildcardServiceExport
// is the subset-match fallback and returns nil when nothing covers.
func TestDetail05(t *testing.T) {
	exp := NewAccount("EXP")
	exp.AddServiceExport("svc.*", nil)
	exp.AddServiceExport("lit", nil)

	exp.mu.Lock()
	defer exp.mu.Unlock()
	if exp.getServiceExport("svc.one") == nil {
		t.Fatal("wildcard export should cover svc.one")
	}
	if exp.getServiceExport("svc.one.two") != nil {
		t.Fatal("svc.* must not cover svc.one.two")
	}
	if exp.getServiceExport("lit") == nil {
		t.Fatal("exact map hit for lit")
	}
	if exp.getServiceExport("lit.x") != nil {
		t.Fatal("literal export must not cover lit.x")
	}
	if exp.getWildcardServiceExport("svc.two") == nil {
		t.Fatal("wildcard helper should match svc.two")
	}
	if exp.getWildcardServiceExport("other.two") != nil {
		t.Fatal("wildcard helper must not match uncovered subject")
	}
}

// TestDetail06: isRevoked — empty map never revokes; revoked iff an entry
// for the subject or jwt.All exists with timestamp >= issuedAt.
func TestDetail06(t *testing.T) {
	eaBbWant(t, isRevoked(nil, "SUBJ", 100), false, "nil map")
	eaBbWant(t, isRevoked(map[string]int64{}, "SUBJ", 100), false, "empty map")
	eaBbWant(t, isRevoked(map[string]int64{"SUBJ": 200}, "SUBJ", 100), true, "newer revocation revokes")
	eaBbWant(t, isRevoked(map[string]int64{"SUBJ": 100}, "SUBJ", 100), true, "equal timestamp revokes")
	eaBbWant(t, isRevoked(map[string]int64{"SUBJ": 50}, "SUBJ", 100), false, "stale revocation does not count")
	eaBbWant(t, isRevoked(map[string]int64{"OTHER": 200}, "SUBJ", 100), false, "unrelated entry does not revoke")
	eaBbWant(t, isRevoked(map[string]int64{jwt.All: 200}, "SUBJ", 100), true, "jwt.All revokes any subject")
	eaBbWant(t, isRevoked(map[string]int64{jwt.All: 50}, "SUBJ", 100), false, "stale jwt.All does not revoke")
}

// TestDetail07: checkUserRevoked is an RLock wrapper over isRevoked on
// a.usersRevoked.
func TestDetail07(t *testing.T) {
	a := NewAccount("ACC")
	eaBbWant(t, a.checkUserRevoked("UKEY", 100), false, "no revocations")
	a.usersRevoked = map[string]int64{"UKEY": 200}
	eaBbWant(t, a.checkUserRevoked("UKEY", 100), true, "newer revocation revokes user")
	eaBbWant(t, a.checkUserRevoked("UKEY", 300), false, "issuance after revocation is valid")
	eaBbWant(t, a.checkUserRevoked("OTHER", 100), false, "unrelated user not revoked")
}

// TestDetail08: the public check*ImportAuthorized wrappers take the account
// read lock and delegate to the NoLock forms; results are identical when the
// caller holds the lock.
func TestDetail08(t *testing.T) {
	exp, imp := NewAccount("EXP"), NewAccount("IMP")
	exp.AddStreamExport("foo.*", nil)
	exp.AddServiceExport("svc.*", nil)

	for _, subj := range []string{"foo.a", "foo.a.b", "bar.x"} {
		exp.mu.RLock()
		got := exp.checkStreamImportAuthorizedNoLock(imp, subj, nil)
		exp.mu.RUnlock()
		eaBbWant(t, got, exp.checkStreamImportAuthorized(imp, subj, nil), "stream wrapper==NoLock for "+subj)
	}
	for _, subj := range []string{"svc.a", "svc.a.b", "other"} {
		exp.mu.RLock()
		got := exp.checkServiceImportAuthorizedNoLock(imp, subj, nil)
		exp.mu.RUnlock()
		eaBbWant(t, got, exp.checkServiceImportAuthorized(imp, subj, nil), "service wrapper==NoLock for "+subj)
	}
}
