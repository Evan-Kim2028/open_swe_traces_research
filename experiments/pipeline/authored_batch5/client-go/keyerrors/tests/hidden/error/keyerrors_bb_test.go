package error

import (
	"fmt"
	"strings"
	"testing"

	"example.internal/kvstore/v2/util"
	"github.com/pingcap/failpoint"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	pkgerrors "github.com/pkg/errors"
)

// Hidden suite for unit keyerrors. One TestDetailNN per DETAILS.md line.

// TestDetail01: ExtractKeyErr maps each populated KeyError field to its
// committed error type; an empty KeyError still yields a non-nil error.
// The cross-field priority order is internal and not asserted.
func TestDetail01(t *testing.T) {
	err := ExtractKeyErr(&kvrpcpb.KeyError{Conflict: &kvrpcpb.WriteConflict{Key: []byte("k")}})
	var wc *ErrWriteConflict
	if err == nil || !pkgerrors.As(err, &wc) {
		t.Fatalf("Conflict did not yield *ErrWriteConflict: %v", err)
	}
	err = ExtractKeyErr(&kvrpcpb.KeyError{Retryable: "retry me"})
	var re *ErrRetryable
	if err == nil || !pkgerrors.As(err, &re) {
		t.Fatalf("Retryable did not yield *ErrRetryable: %v", err)
	}
	err = ExtractKeyErr(&kvrpcpb.KeyError{AssertionFailed: &kvrpcpb.AssertionFailed{StartTs: 1}})
	var af *ErrAssertionFailed
	if err == nil || !pkgerrors.As(err, &af) {
		t.Fatalf("AssertionFailed did not yield *ErrAssertionFailed: %v", err)
	}
	for name, ke := range map[string]*kvrpcpb.KeyError{
		"abort":            {Abort: "aborted"},
		"commitTsTooLarge": {CommitTsTooLarge: &kvrpcpb.CommitTsTooLarge{}},
		"txnNotFound":      {TxnNotFound: &kvrpcpb.TxnNotFound{}},
		"empty":            {},
	} {
		if ExtractKeyErr(ke) == nil {
			t.Fatalf("%s: nil error", name)
		}
	}
}

// TestDetail02: the mockRetryableErrorResp failpoint rewrites the KeyError
// before dispatch — a Conflict input comes back retryable.
func TestDetail02(t *testing.T) {
	util.EnableFailpoints()
	if err := failpoint.Enable("tikvclient/mockRetryableErrorResp", "return(true)"); err != nil {
		t.Fatalf("enable failpoint: %v", err)
	}
	defer failpoint.Disable("tikvclient/mockRetryableErrorResp")
	err := ExtractKeyErr(&kvrpcpb.KeyError{Conflict: &kvrpcpb.WriteConflict{Key: []byte("k")}})
	var re *ErrRetryable
	if err == nil || !pkgerrors.As(err, &re) {
		t.Fatalf("failpoint did not rewrite to retryable: %v", err)
	}
}

// TestDetail03: the predicates unwrap, so wrapped errors match.
func TestDetail03(t *testing.T) {
	wc := &ErrWriteConflict{WriteConflict: &kvrpcpb.WriteConflict{Key: []byte("k")}}
	if !IsErrWriteConflict(fmt.Errorf("outer: %w", wc)) {
		t.Fatalf("IsErrWriteConflict missed wrapped error")
	}
	ke := &ErrKeyExist{AlreadyExist: &kvrpcpb.AlreadyExist{Key: []byte("k")}}
	if !IsErrKeyExist(fmt.Errorf("outer: %w", ke)) {
		t.Fatalf("IsErrKeyExist missed wrapped error")
	}
	if !IsErrorUndetermined(fmt.Errorf("outer: %w", ErrResultUndetermined)) {
		t.Fatalf("IsErrorUndetermined missed wrapped error")
	}
	if IsErrWriteConflict(fmt.Errorf("outer: %w", ke)) || IsErrKeyExist(fmt.Errorf("outer: %w", wc)) {
		t.Fatalf("predicates crossed")
	}
}

// TestDetail04: Error() strings embed the wrapped proto or the stored value.
func TestDetail04(t *testing.T) {
	pb := &kvrpcpb.WriteConflict{Key: []byte{0xab}}
	e := &ErrWriteConflict{WriteConflict: pb}
	if !strings.Contains(e.Error(), fmt.Sprintf("%v", pb)) {
		t.Fatalf("ErrWriteConflict.Error() = %q does not embed the proto", e.Error())
	}
	afb := &kvrpcpb.AssertionFailed{StartTs: 9}
	ae := &ErrAssertionFailed{AssertionFailed: afb}
	if !strings.Contains(ae.Error(), fmt.Sprintf("%v", afb)) {
		t.Fatalf("ErrAssertionFailed.Error() = %q does not embed the proto", ae.Error())
	}
	if got := (&ErrTxnTooLarge{Size: 42}).Error(); !strings.Contains(got, "42") {
		t.Fatalf("ErrTxnTooLarge.Error() = %q does not embed the size", got)
	}
}

// TestDetail05: NewErrWriteConflictWithArgs packs the args into the proto.
func TestDetail05(t *testing.T) {
	e := NewErrWriteConflictWithArgs(11, 22, 33, []byte("key"), kvrpcpb.WriteConflict_Optimistic)
	if e == nil || e.WriteConflict == nil {
		t.Fatalf("nil result")
	}
	if e.StartTs != 11 || e.ConflictTs != 22 || e.ConflictCommitTs != 33 ||
		string(e.Key) != "key" || e.Reason != kvrpcpb.WriteConflict_Optimistic {
		t.Fatalf("packed fields: %+v", e.WriteConflict)
	}
}

// TestDetail06: ErrRetryable.Error returns the raw string;
// ErrPDServerTimeout.Error returns the stored msg via the error interface.
func TestDetail06(t *testing.T) {
	if got := (&ErrRetryable{Retryable: "boom"}).Error(); got != "boom" {
		t.Fatalf("ErrRetryable.Error() = %q", got)
	}
	var err error = NewErrPDServerTimeout("pd slow")
	if err == nil || err.Error() != "pd slow" {
		t.Fatalf("NewErrPDServerTimeout().Error() = %v", err)
	}
}
