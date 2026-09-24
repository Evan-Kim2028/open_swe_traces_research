// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v92/github"
)

// signServer stands up a client whose API host records the create-commit
// request body. It returns the client and a pointer to the last body seen.
func signServer(t *testing.T) (*github.Client, *[]byte) {
	t.Helper()
	var got []byte
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/git/commits", func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"sha":"newcommit"}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client, err := github.NewClient(github.WithURLs(github.Ptr(srv.URL+"/"), github.Ptr(srv.URL+"/")))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client, &got
}

func signedCommit() github.Commit {
	return github.Commit{
		Message: github.Ptr("commit subject\n\ncommit body"),
		Author: &github.CommitAuthor{
			Name:  github.Ptr("A U Thor"),
			Email: github.Ptr("a@b.c"),
			Date:  &github.Timestamp{Time: time.Unix(1700000000, 0)},
		},
		Tree:    &github.Tree{SHA: github.Ptr("treesha")},
		Parents: []*github.Commit{{SHA: github.Ptr("p1")}, {SHA: github.Ptr("p2")}},
	}
}

// Detail 1: a nil signer is rejected with an error. (Inferable: yes —
// defensive boundary. A typed-nil MessageSignerFunc is the only nil-valued
// signer reachable through the public option.)
func TestDetail01(t *testing.T) {
	client, _ := signServer(t)
	opts := &github.CreateCommitOptions{Signer: github.MessageSignerFunc(nil)}
	_, _, err := client.Git.CreateCommit(context.Background(), "o", "r", signedCommit(), opts)
	if err == nil {
		t.Fatal("nil signer was accepted")
	}
}

// Detail 2: the signed payload is the Git object text: "tree <sha>", one
// "parent <sha>" per parent, "author <name> <<email>> <unix> <±HHMM>", the
// same for committer, a blank line, then the message. Inferable: partially —
// the field order and labels are the documented object convention; asserted
// as structure, not byte-exact spacing beyond that convention.
func TestDetail02(t *testing.T) {
	client, _ := signServer(t)

	var payload []byte
	signer := github.MessageSignerFunc(func(w io.Writer, r io.Reader) error {
		payload, _ = io.ReadAll(r)
		_, err := w.Write([]byte("SIGNATURE"))
		return err
	})
	opts := &github.CreateCommitOptions{Signer: signer}
	if _, _, err := client.Git.CreateCommit(context.Background(), "o", "r", signedCommit(), opts); err != nil {
		t.Fatalf("CreateCommit: %v", err)
	}
	if len(payload) == 0 {
		t.Fatal("signer was never given a payload")
	}
	p := string(payload)

	if !strings.HasPrefix(p, "tree treesha\n") {
		t.Errorf("payload does not start with the tree line: %q", p)
	}
	if !strings.Contains(p, "\nparent p1\nparent p2\n") {
		t.Errorf("payload lacks ordered parent lines: %q", p)
	}
	authorRe := regexp.MustCompile(`(?m)^author A U Thor <a@b\.c> 1700000000 [-+]\d{4}$`)
	if !authorRe.MatchString(p) {
		t.Errorf("payload lacks the author line shape: %q", p)
	}
	committerRe := regexp.MustCompile(`(?m)^committer [^\n]+ <[^>]+> \d+ [-+]\d{4}$`)
	if !committerRe.MatchString(p) {
		t.Errorf("payload lacks the committer line shape: %q", p)
	}
	if !strings.Contains(p, "\n\ncommit subject\n\ncommit body") {
		t.Errorf("payload lacks blank-line separator before the message: %q", p)
	}
}

// Detail 3: a nil committer falls back to the author. (Inferable: doc)
func TestDetail03(t *testing.T) {
	client, _ := signServer(t)

	var payload []byte
	signer := github.MessageSignerFunc(func(w io.Writer, r io.Reader) error {
		payload, _ = io.ReadAll(r)
		_, err := w.Write([]byte("SIGNATURE"))
		return err
	})
	opts := &github.CreateCommitOptions{Signer: signer}
	c := signedCommit()
	c.Committer = nil
	if _, _, err := client.Git.CreateCommit(context.Background(), "o", "r", c, opts); err != nil {
		t.Fatalf("CreateCommit: %v", err)
	}

	p := string(payload)
	var authorTail, committerTail string
	for _, line := range strings.Split(p, "\n") {
		if rest, ok := strings.CutPrefix(line, "author "); ok {
			authorTail = rest
		}
		if rest, ok := strings.CutPrefix(line, "committer "); ok {
			committerTail = rest
		}
	}
	if authorTail == "" || committerTail == "" {
		t.Fatalf("payload lacks author/committer lines: %q", p)
	}
	if authorTail != committerTail {
		t.Errorf("nil committer did not fall back to author: author=%q committer=%q", authorTail, committerTail)
	}
}

// Detail 4: a commit missing required fields is rejected before any
// serialization — the signer is never invoked. Inferable: partially — the
// committed missing-field cases (nil/empty message, nil author) are
// asserted; the exact field set is not pinned beyond them.
func TestDetail04(t *testing.T) {
	client, got := signServer(t)

	called := false
	signer := github.MessageSignerFunc(func(w io.Writer, r io.Reader) error {
		called = true
		return nil
	})
	opts := &github.CreateCommitOptions{Signer: signer}

	cases := map[string]github.Commit{
		"empty commit": {},
		"empty message": func() github.Commit {
			c := signedCommit()
			c.Message = github.Ptr("")
			return c
		}(),
		"nil author": func() github.Commit {
			c := signedCommit()
			c.Author = nil
			return c
		}(),
	}
	for name, c := range cases {
		called = false
		*got = nil
		if _, _, err := client.Git.CreateCommit(context.Background(), "o", "r", c, opts); err == nil {
			t.Errorf("%s: expected error, got none", name)
		}
		if called {
			t.Errorf("%s: signer invoked for an invalid commit", name)
		}
	}
}

// Detail 5: the signature attached to the request is exactly what the signer
// wrote for the payload — a streamed token, not a reconstruction.
// (Inferable: yes)
func TestDetail05(t *testing.T) {
	client, got := signServer(t)

	sig := "-----BEGIN-----\nabc123\n-----END-----"
	signer := github.MessageSignerFunc(func(w io.Writer, r io.Reader) error {
		io.Copy(io.Discard, r)
		_, err := w.Write([]byte(sig))
		return err
	})
	opts := &github.CreateCommitOptions{Signer: signer}
	if _, _, err := client.Git.CreateCommit(context.Background(), "o", "r", signedCommit(), opts); err != nil {
		t.Fatalf("CreateCommit: %v", err)
	}
	if len(*got) == 0 {
		t.Fatal("server saw no request body")
	}
	var body struct {
		Signature *string `json:"signature"`
	}
	if err := json.Unmarshal(*got, &body); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if body.Signature == nil || *body.Signature != sig {
		t.Errorf("signature = %v, want the signer's exact output", body.Signature)
	}
}
