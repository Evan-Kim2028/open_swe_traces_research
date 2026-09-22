package github

import (
	"errors"
	"net/http"
	"testing"
)

// TestDetail01: on *AcceptedError the raw body is still decoded into the
// returned *Repository — the pending fork's fields are populated alongside
// the error.
func TestDetail01(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/forks", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"id":1,"name":"pending-fork"}`))
	})

	fork, _, err := client.Repositories.CreateFork(t.Context(), "o", "r", nil)
	var ae *AcceptedError
	if !errors.As(err, &ae) {
		t.Fatalf("err = %v (%T), want *AcceptedError", err, err)
	}
	if fork == nil || fork.GetName() != "pending-fork" {
		t.Fatalf("pending fork not decoded from 202 body: %+v", fork)
	}
}

// TestDetail02: a 202 whose body does not decode still returns the
// *AcceptedError — the decode failure does not replace it.
func TestDetail02(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/forks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{not json`))
	})

	fork, _, err := client.Repositories.CreateFork(t.Context(), "o", "r", nil)
	var ae *AcceptedError
	if !errors.As(err, &ae) {
		t.Fatalf("err = %v (%T), want *AcceptedError", err, err)
	}
	if fork != nil {
		t.Fatalf("fork = %+v, want nil on undecodable 202 body", fork)
	}
}

// TestDetail03: a non-accepted response behaves normally — repository
// populated on success, error propagated on failure.
func TestDetail03(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/repos/o/r/forks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":2,"name":"forked"}`))
	})
	mux.HandleFunc("/repos/o/missing/forks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	})

	fork, _, err := client.Repositories.CreateFork(t.Context(), "o", "r", nil)
	if err != nil {
		t.Fatalf("CreateFork 200: %v", err)
	}
	if fork == nil || fork.GetName() != "forked" {
		t.Fatalf("CreateFork = %+v, want name=forked", fork)
	}

	fork, _, err = client.Repositories.CreateFork(t.Context(), "o", "missing", nil)
	if err == nil {
		t.Fatal("CreateFork 404: err = nil")
	}
	var ae *AcceptedError
	if errors.As(err, &ae) {
		t.Fatal("404 surfaced as *AcceptedError")
	}
}
