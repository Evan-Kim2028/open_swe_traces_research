package github

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
)

// bbReadFailBody fails every Read — simulates a mid-body transport error.
type bbReadFailBody struct{}

func (bbReadFailBody) Read([]byte) (int, error) { return 0, errors.New("bb mid-body read failure") }
func (bbReadFailBody) Close() error             { return nil }

type bbReadFailRT struct{}

func (bbReadFailRT) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       bbReadFailBody{},
		Request:    req,
	}, nil
}

// TestDetail01: an empty or whitespace-only body produces no decode error for
// a non-nil, non-io.Writer target.
func TestDetail01(t *testing.T) {
	t.Parallel()
	for _, body := range []string{"", "   \n\t  "} {
		client, mux, _ := setup(t)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(body))
		})
		req, err := client.NewRequest(t.Context(), "GET", ".", nil)
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		v := new(struct {
			Name *string `json:"name,omitempty"`
		})
		if _, err := client.Do(req, v); err != nil {
			t.Fatalf("Do with body %q: err = %v, want nil", body, err)
		}
	}
}

// TestDetail02: a non-empty body that fails JSON decode surfaces the decode
// error.
func TestDetail02(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not json`))
	})
	req, err := client.NewRequest(t.Context(), "GET", ".", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, err := client.Do(req, new(map[string]any)); err == nil {
		t.Fatal("Do with undecodable body: err = nil, want decode error")
	}
}

// TestDetail03: a body that cannot be read surfaces the read error.
func TestDetail03(t *testing.T) {
	t.Parallel()
	client, _, _ := setup(t)
	client.client = &http.Client{Transport: bbReadFailRT{}}

	req, err := client.NewRequest(t.Context(), "GET", ".", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, err := client.Do(req, new(map[string]any)); err == nil {
		t.Fatal("Do with unreadable body: err = nil, want read error")
	}
}

// TestDetail04: io.Writer targets receive the raw body bytes with no JSON
// decode.
func TestDetail04(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	const body = `raw not json {`
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})
	req, err := client.NewRequest(t.Context(), "GET", ".", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	var buf bytes.Buffer
	if _, err := client.Do(req, &buf); err != nil {
		t.Fatalf("Do into io.Writer: %v", err)
	}
	if buf.String() != body {
		t.Fatalf("io.Writer got %q, want %q", buf.String(), body)
	}
}

// TestDetail05: a nil target performs no decode at all — an undecodable body
// is not an error.
func TestDetail05(t *testing.T) {
	t.Parallel()
	client, mux, _ := setup(t)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not json`))
	})
	req, err := client.NewRequest(t.Context(), "GET", ".", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := client.Do(req, nil)
	if err != nil {
		t.Fatalf("Do with nil target: %v", err)
	}
	if resp == nil {
		t.Fatal("Do with nil target returned nil *Response")
	}
	_, _ = io.Copy(io.Discard, resp.Body)
}
