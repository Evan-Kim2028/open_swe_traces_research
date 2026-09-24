package github

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func bbCommitBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("request body not JSON: %v", err)
	}
	return m
}

func bbValidCommit() Commit {
	return Commit{
		Message: new(string),
		Author: &CommitAuthor{
			Name:  new(string),
			Email: new(string),
			Date:  &Timestamp{Time: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)},
		},
	}
}

// TestDetail01: nil opts is legal — the call proceeds with default options.
func TestDetail01(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/git/commits", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"sha":"abc"}`))
	})

	c := bbValidCommit()
	got, _, err := client.Git.CreateCommit(t.Context(), "o", "r", c, nil)
	if err != nil {
		t.Fatalf("CreateCommit with nil opts: %v", err)
	}
	if got == nil || got.GetSHA() != "abc" {
		t.Fatalf("CreateCommit = %+v, want sha=abc", got)
	}
}

// TestDetail02: a stored Verification.Signature is copied into the request
// body and no signer is invoked.
func TestDetail02(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var gotSig atomic.Value
	mux.HandleFunc("/repos/o/r/git/commits", func(w http.ResponseWriter, r *http.Request) {
		gotSig.Store(bbCommitBody(t, r)["signature"])
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	var signerCalls int32
	signer := MessageSignerFunc(func(w io.Writer, r io.Reader) error {
		atomic.AddInt32(&signerCalls, 1)
		_, _ = w.Write([]byte("signer-sig"))
		return nil
	})

	c := bbValidCommit()
	*c.Message = "commit message"
	c.Verification = &SignatureVerification{Signature: new(string)}
	*c.Verification.Signature = "stored-sig"

	if _, _, err := client.Git.CreateCommit(t.Context(), "o", "r", c, &CreateCommitOptions{Signer: signer}); err != nil {
		t.Fatalf("CreateCommit: %v", err)
	}
	if v := gotSig.Load(); v != "stored-sig" {
		t.Fatalf("request signature = %v, want stored-sig", v)
	}
	if n := atomic.LoadInt32(&signerCalls); n != 0 {
		t.Fatalf("signer invoked %d times despite stored signature", n)
	}
}

// TestDetail03: with no Verification and a Signer set, the body's Signature
// comes from running the signer over the commit content.
func TestDetail03(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var gotSig atomic.Value
	mux.HandleFunc("/repos/o/r/git/commits", func(w http.ResponseWriter, r *http.Request) {
		gotSig.Store(bbCommitBody(t, r)["signature"])
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	var sawInput int32
	signer := MessageSignerFunc(func(w io.Writer, r io.Reader) error {
		b, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		if len(b) == 0 {
			return errors.New("empty canonical input")
		}
		atomic.AddInt32(&sawInput, 1)
		_, _ = w.Write([]byte("generated-sig"))
		return nil
	})

	c := bbValidCommit()
	*c.Message = "commit message"

	if _, _, err := client.Git.CreateCommit(t.Context(), "o", "r", c, &CreateCommitOptions{Signer: signer}); err != nil {
		t.Fatalf("CreateCommit: %v", err)
	}
	if v := gotSig.Load(); v != "generated-sig" {
		t.Fatalf("request signature = %v, want generated-sig", v)
	}
	if atomic.LoadInt32(&sawInput) == 0 {
		t.Fatal("signer never invoked")
	}
}

// TestDetail04: signer errors abort the call before any request is built.
func TestDetail04(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)

	var hits int32
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	})

	signer := MessageSignerFunc(func(w io.Writer, r io.Reader) error {
		return errors.New("signer boom")
	})

	c := bbValidCommit()
	*c.Message = "commit message"

	if _, _, err := client.Git.CreateCommit(t.Context(), "o", "r", c, &CreateCommitOptions{Signer: signer}); err == nil {
		t.Fatal("CreateCommit with failing signer: err = nil")
	}
	if n := atomic.LoadInt32(&hits); n != 0 {
		t.Fatalf("request built despite signer failure: %d hits", n)
	}
}
