// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-github/v92/github"
)

// hostedRunnerServer returns a client plus a hit counter for the hosted
// runner creation endpoints.
func hostedRunnerServer(t *testing.T) (*github.Client, *atomic.Int64) {
	t.Helper()
	var hits atomic.Int64
	mux := http.NewServeMux()
	handler := func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":1}`)
	}
	mux.HandleFunc("/orgs/o/actions/hosted-runners", handler)
	mux.HandleFunc("/enterprises/e/actions/hosted-runners", handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	base := srv.URL + "/"
	client, err := github.NewClient(github.WithURLs(&base, &base))
	if err != nil {
		t.Fatalf("github.NewClient returned error: %v", err)
	}
	return client, &hits
}

func validRunnerReq() github.CreateHostedRunnerRequest {
	return github.CreateHostedRunnerRequest{
		Name:          "runner-1",
		Image:         github.HostedRunnerImage{ID: "img-1", Source: "github"},
		Size:          "4-core",
		RunnerGroupID: 7,
	}
}

// Detail 1: an empty Name is rejected before any network call. (Inferable:
// doc)
func TestDetail01(t *testing.T) {
	client, hits := hostedRunnerServer(t)
	req := validRunnerReq()
	req.Name = ""
	_, _, err := client.Actions.CreateHostedRunner(context.Background(), "o", req)
	if err == nil {
		t.Fatal("empty Name: CreateHostedRunner returned nil error")
	}
	if hits.Load() != 0 {
		t.Error("empty-Name request reached the network")
	}
}

// Detail 2: a zero-value Image is rejected. (Inferable: doc)
func TestDetail02(t *testing.T) {
	client, hits := hostedRunnerServer(t)
	req := validRunnerReq()
	req.Image = github.HostedRunnerImage{}
	_, _, err := client.Actions.CreateHostedRunner(context.Background(), "o", req)
	if err == nil {
		t.Fatal("zero Image: CreateHostedRunner returned nil error")
	}
	if hits.Load() != 0 {
		t.Error("zero-Image request reached the network")
	}
}

// Detail 3: an empty Size is rejected. (Inferable: doc)
func TestDetail03(t *testing.T) {
	client, hits := hostedRunnerServer(t)
	req := validRunnerReq()
	req.Size = ""
	_, _, err := client.Actions.CreateHostedRunner(context.Background(), "o", req)
	if err == nil {
		t.Fatal("empty Size: CreateHostedRunner returned nil error")
	}
	if hits.Load() != 0 {
		t.Error("empty-Size request reached the network")
	}
}

// Detail 4: a zero RunnerGroupID is rejected. (Inferable: doc)
func TestDetail04(t *testing.T) {
	client, hits := hostedRunnerServer(t)
	req := validRunnerReq()
	req.RunnerGroupID = 0
	_, _, err := client.Actions.CreateHostedRunner(context.Background(), "o", req)
	if err == nil {
		t.Fatal("zero RunnerGroupID: CreateHostedRunner returned nil error")
	}
	if hits.Load() != 0 {
		t.Error("zero-RunnerGroupID request reached the network")
	}
}

// Detail 5: when all four fields are present the request is accepted and the
// call proceeds to the API. (Inferable: yes)
func TestDetail05(t *testing.T) {
	client, hits := hostedRunnerServer(t)
	_, _, err := client.Actions.CreateHostedRunner(context.Background(), "o", validRunnerReq())
	if err != nil {
		t.Fatalf("valid request: CreateHostedRunner returned error: %v", err)
	}
	if hits.Load() != 1 {
		t.Errorf("valid request did not reach the server (hits=%d)", hits.Load())
	}

	// The enterprise variant shares the validator.
	_, _, err = client.Enterprise.CreateHostedRunner(context.Background(), "e", validRunnerReq())
	if err != nil {
		t.Fatalf("valid enterprise request: CreateHostedRunner returned error: %v", err)
	}
}

// Detail 6: each rejection produces a field-specific error message.
// Inferable: no — asserted as shape only: the surfaced error names the
// offending field; the literal message is never asserted.
func TestDetail06(t *testing.T) {
	client, _ := hostedRunnerServer(t)
	ctx := context.Background()
	cases := []struct {
		name  string
		mut   func(*github.CreateHostedRunnerRequest)
		field string
	}{
		{"name", func(r *github.CreateHostedRunnerRequest) { r.Name = "" }, "name"},
		{"image", func(r *github.CreateHostedRunnerRequest) { r.Image = github.HostedRunnerImage{} }, "image"},
		{"size", func(r *github.CreateHostedRunnerRequest) { r.Size = "" }, "size"},
		{"group", func(r *github.CreateHostedRunnerRequest) { r.RunnerGroupID = 0 }, "runner"},
	}
	for _, c := range cases {
		req := validRunnerReq()
		c.mut(&req)
		_, _, err := client.Actions.CreateHostedRunner(ctx, "o", req)
		if err == nil {
			t.Errorf("%s missing: nil error", c.name)
			continue
		}
		if !strings.Contains(strings.ToLower(err.Error()), c.field) {
			t.Errorf("%s missing: error %q does not name the offending field", c.name, err.Error())
		}
	}
}
