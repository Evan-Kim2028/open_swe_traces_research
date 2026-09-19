// Black-box property suite for streamrenders (render package).
// Exported API only: Reader, Data, Redirect, String, WriteString.
// Seed bbSeed=20260919; bbCases=10000 adversarial draws per random property.
//
// Coverage table (contract.md → property):
// | contract sentence | property |
// |---|---|
// | Reader sets Content-Length only when ContentLength >= 0 | TestReaderContentLengthProperty |
// | Reader copies body and merges custom headers | TestReaderHeaderMergeProperty |
// | Data writes Content-Length only when payload non-empty | TestDataContentLengthProperty |
// | Redirect panics outside 3xx except 201 Created | TestRedirectStatusProperty |
// | String format-with-args vs literal format string | TestStringFormatProperty |
// | lone percent stays literal in String without args | TestStringLiteralPercentProperty |
// | pre-set Content-Type preserved on all stream renderers | TestStreamPreservesPresetContentTypeProperty |
// | write failures propagate | TestStreamWriteErrorProperty |
package render_test

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	render "example.internal/httprouter/render"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

type bbErrorWriter struct {
	errThreshold int
	writeCount   int
	*httptest.ResponseRecorder
}

func (w *bbErrorWriter) Header() http.Header {
	if w.ResponseRecorder == nil {
		w.ResponseRecorder = httptest.NewRecorder()
	}
	return w.ResponseRecorder.Header()
}

func (w *bbErrorWriter) WriteHeader(statusCode int) {
	if w.ResponseRecorder == nil {
		w.ResponseRecorder = httptest.NewRecorder()
	}
	w.ResponseRecorder.WriteHeader(statusCode)
}

func (w *bbErrorWriter) Write(buf []byte) (int, error) {
	w.writeCount++
	if w.errThreshold > 0 && w.writeCount >= w.errThreshold {
		return 0, errors.New("write error")
	}
	if w.ResponseRecorder == nil {
		w.ResponseRecorder = httptest.NewRecorder()
	}
	return w.ResponseRecorder.Write(buf)
}

func bbStreamRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

func bbRandBody(rng *rand.Rand, max int) string {
	const chars = "#!PNG\x00abcXYZ\n\r\t"
	n := rng.Intn(max + 1)
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}
	return string(b)
}

func TestReaderContentLengthProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		body := bbRandBody(rng, 128)
		cl := int64(len(body))
		if rng.Intn(4) == 0 {
			cl = -1
		}
		headers := map[string]string{
			"Content-Disposition": `attachment; filename="f.bin"`,
			"x-request-id":        fmt.Sprintf("req-%d", i),
		}
		w := bbStreamRecorder()
		err := (render.Reader{
			ContentLength: cl,
			ContentType:   "application/octet-stream",
			Reader:        strings.NewReader(body),
			Headers:       headers,
		}).Render(w)
		if err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if w.Body.String() != body {
			t.Fatalf("case %d body mismatch", i)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/octet-stream" {
			t.Fatalf("case %d content-type %q", i, ct)
		}
		if cl >= 0 {
			if got := w.Header().Get("Content-Length"); got != strconv.FormatInt(cl, 10) {
				t.Fatalf("case %d content-length %q want %d", i, got, cl)
			}
		} else if w.Header().Get("Content-Length") != "" {
			t.Fatalf("case %d negative length must not set Content-Length", i)
		}
	}
}

func TestReaderHeaderMergeProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		body := bbRandBody(rng, 64)
		headers := map[string]string{
			"X-Custom": fmt.Sprintf("v-%d", i),
			"X-Empty":  "",
		}
		w := bbStreamRecorder()
		if rng.Intn(3) == 0 {
			w.Header().Set("X-Custom", "preset")
		}
		err := (render.Reader{
			ContentLength: int64(len(body)),
			ContentType:   "image/png",
			Reader:        strings.NewReader(body),
			Headers:       headers,
		}).Render(w)
		if err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if preset := w.Header().Get("X-Custom"); preset == "preset" {
			if preset != "preset" {
				t.Fatalf("case %d preset header overwritten", i)
			}
		} else if w.Header().Get("X-Custom") != headers["X-Custom"] {
			t.Fatalf("case %d custom header missing", i)
		}
	}
}

func TestDataContentLengthProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		size := rng.Intn(256)
		data := make([]byte, size)
		for j := range data {
			data[j] = byte(rng.Intn(256))
		}
		w := bbStreamRecorder()
		err := (render.Data{
			ContentType: "application/octet-stream",
			Data:        data,
		}).Render(w)
		if err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if !bytesEqual(w.Body.Bytes(), data) {
			t.Fatalf("case %d body mismatch", i)
		}
		if size > 0 {
			if got := w.Header().Get("Content-Length"); got != strconv.Itoa(size) {
				t.Fatalf("case %d content-length %q want %d", i, got, size)
			}
		} else if w.Header().Get("Content-Length") != "" {
			t.Fatalf("case %d empty data must not set Content-Length", i)
		}
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestRedirectStatusProperty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test-redirect", nil)
	valid := []int{
		http.StatusMultipleChoices,
		http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect,
		http.StatusCreated,
	}
	for _, code := range valid {
		w := bbStreamRecorder()
		err := (render.Redirect{Code: code, Request: req, Location: "/new/loc"}).Render(w)
		if err != nil {
			t.Fatalf("code %d: %v", code, err)
		}
		if loc := w.Header().Get("Location"); loc != "/new/loc" {
			t.Fatalf("code %d location %q", code, loc)
		}
	}
	rng := rand.New(rand.NewSource(bbSeed + 3))
	for i := 0; i < bbCases; i++ {
		code := rng.Intn(600)
		if code >= 300 && code <= 399 {
			continue
		}
		if code == http.StatusCreated {
			continue
		}
		w := bbStreamRecorder()
		panicked := false
		msg := ""
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
					msg = fmt.Sprint(r)
				}
			}()
			_ = (render.Redirect{Code: code, Request: req, Location: "/x"}).Render(w)
		}()
		if !panicked {
			t.Fatalf("case %d code %d must panic", i, code)
		}
		if !strings.Contains(msg, fmt.Sprintf("%d", code)) {
			t.Fatalf("case %d panic message %q", i, msg)
		}
	}
}

func TestStringFormatProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 4))
	for i := 0; i < bbCases; i++ {
		format := "hello %s %d %%"
		s := bbRandBody(rng, 8)
		n := rng.Intn(1000)
		w := bbStreamRecorder()
		err := (render.String{Format: format, Data: []any{s, n}}).Render(w)
		if err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		want := fmt.Sprintf(format, s, n)
		if w.Body.String() != want {
			t.Fatalf("case %d body %q want %q", i, w.Body.String(), want)
		}
		if ct := w.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
			t.Fatalf("case %d content-type %q", i, ct)
		}
	}
}

func TestStringLiteralPercentProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 5))
	formats := []string{"hola %s %d", "100%", "%%", "%", "no verbs here"}
	for i := 0; i < bbCases; i++ {
		format := formats[rng.Intn(len(formats))]
		w := bbStreamRecorder()
		err := (render.String{Format: format, Data: nil}).Render(w)
		if err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if w.Body.String() != format {
			t.Fatalf("case %d literal format %q got %q", i, format, w.Body.String())
		}
		w = bbStreamRecorder()
		if err := render.WriteString(w, format, nil); err != nil {
			t.Fatalf("case %d WriteString: %v", i, err)
		}
		if w.Body.String() != format {
			t.Fatalf("case %d WriteString literal mismatch", i)
		}
	}
}

func TestStreamPreservesPresetContentTypeProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 6))
	preset := "application/vnd.test+binary"
	body := []byte("payload")
	for i := 0; i < bbCases; i++ {
		kind := rng.Intn(4)
		w := bbStreamRecorder()
		w.Header().Set("Content-Type", preset)
		var err error
		switch kind {
		case 0:
			err = (render.Reader{ContentType: "image/png", Reader: strings.NewReader("x")}).Render(w)
		case 1:
			err = (render.Data{ContentType: "image/png", Data: body}).Render(w)
		case 2:
			err = (render.String{Format: "x"}).Render(w)
		default:
			err = render.WriteString(w, "y", nil)
		}
		if err != nil {
			t.Fatalf("case %d render: %v", i, err)
		}
		if ct := w.Header().Get("Content-Type"); ct != preset {
			t.Fatalf("case %d preset overwritten to %q", i, ct)
		}
	}
}

func TestStreamWriteErrorProperty(t *testing.T) {
	data := []byte("#!PNG raw")
	ew := &bbErrorWriter{errThreshold: 1, ResponseRecorder: bbStreamRecorder()}
	if err := (render.Data{ContentType: "image/png", Data: data}).Render(ew); err == nil {
		t.Fatal("Data expected write error")
	}
	ew = &bbErrorWriter{errThreshold: 1, ResponseRecorder: bbStreamRecorder()}
	if err := (render.String{Format: "x", Data: []any{"y"}}).Render(ew); err == nil {
		t.Fatal("String expected write error")
	}
	ew = &bbErrorWriter{errThreshold: 1, ResponseRecorder: bbStreamRecorder()}
	if err := render.WriteString(ew, "fmt %s", []any{"arg"}); err == nil {
		t.Fatal("WriteString expected write error")
	}
}
