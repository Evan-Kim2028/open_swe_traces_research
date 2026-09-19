// Black-box property suite for the multipartfiles unit.
// Exported API only: binding.FormMultipart, binding.ErrMultiFileHeader,
// binding.ErrMultiFileHeaderLenInvalid, binding.MIMEMultipartPOSTForm.
// Seed 20260919; >=10k cases; contract + multipart_form_mapping_test.go coverage.
//
// Coverage table (contract sentence -> property):
//   "*multipart.FileHeader receives the first uploaded file"
//       -> TestBBMultipartPointerFirstFile / TestBBMultipartPointerRandom
//   "multipart.FileHeader value field receives a copy of the first file header"
//       -> TestBBMultipartValueFirstFile
//   "slice of file-header types rebuilt to exactly the number of uploaded files"
//       -> TestBBMultipartSliceCount / TestBBMultipartSliceRandom
//   "fixed array must equal the file count or binding fails with length error"
//       -> TestBBMultipartArrayExactLen / TestBBMultipartArrayLenInvalid
//   "field kind that cannot hold a file returns exported invalid-field-type error"
//       -> TestBBMultipartWrongTypeError / TestBBMultipartWrongTypeRandom
//   "ordinary form values used when key has no uploaded files"
//       -> TestBBMultipartFormValueFallback
package binding_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"testing"

	binding "example.internal/httprouter/binding"
)

// bbSeed and bbCases are in bb_const_test.go.

type bbMultipartFile struct {
	Field    string
	Filename string
	Content  []byte
}

func bbCreateMultipartRequest(files []bbMultipartFile, fields map[string]string) *http.Request {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	for _, f := range files {
		fw, _ := mw.CreateFormFile(f.Field, f.Filename)
		_, _ = fw.Write(f.Content)
	}
	_ = mw.Close()
	req, _ := http.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", binding.MIMEMultipartPOSTForm+"; boundary="+mw.Boundary())
	return req
}

func bbReadFileHeader(fh *multipart.FileHeader) ([]byte, error) {
	rc, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func TestBBMultipartPointerFirstFile(t *testing.T) {
	file := bbMultipartFile{"file", "one.txt", []byte("hello")}
	req := bbCreateMultipartRequest([]bbMultipartFile{file}, nil)

	var s struct {
		F *multipart.FileHeader `form:"file"`
	}
	if err := binding.FormMultipart.Bind(req, &s); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if s.F == nil {
		t.Fatal("expected non-nil FileHeader pointer")
	}
	if s.F.Filename != file.Filename {
		t.Fatalf("filename=%q want %q", s.F.Filename, file.Filename)
	}
	body, err := bbReadFileHeader(s.F)
	if err != nil || string(body) != string(file.Content) {
		t.Fatalf("content mismatch: %v body=%q", err, body)
	}
}

func TestBBMultipartPointerRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		n := rng.Intn(3) + 1
		files := make([]bbMultipartFile, n)
		for j := range files {
			files[j] = bbMultipartFile{
				"file",
				fmt.Sprintf("f%d.txt", j),
				[]byte(fmt.Sprintf("payload-%d-%d", i, j)),
			}
		}
		req := bbCreateMultipartRequest(files, nil)
		var s struct {
			F *multipart.FileHeader `form:"file"`
		}
		if err := binding.FormMultipart.Bind(req, &s); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if s.F.Filename != files[0].Filename {
			t.Fatalf("case %d: want first file %q got %q", i, files[0].Filename, s.F.Filename)
		}
	}
}

func TestBBMultipartValueFirstFile(t *testing.T) {
	file := bbMultipartFile{"file", "val.txt", []byte("copy-me")}
	req := bbCreateMultipartRequest([]bbMultipartFile{file}, nil)

	var s struct {
		F multipart.FileHeader `form:"file"`
	}
	if err := binding.FormMultipart.Bind(req, &s); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if s.F.Filename != file.Filename {
		t.Fatalf("filename=%q want %q", s.F.Filename, file.Filename)
	}
	body, _ := bbReadFileHeader(&s.F)
	if string(body) != string(file.Content) {
		t.Fatalf("content=%q want %q", body, file.Content)
	}
}

func TestBBMultipartSliceCount(t *testing.T) {
	files := []bbMultipartFile{
		{"file", "a.txt", []byte("a")},
		{"file", "b.txt", []byte("b")},
	}
	req := bbCreateMultipartRequest(files, nil)

	var s struct {
		Files []*multipart.FileHeader `form:"file"`
	}
	if err := binding.FormMultipart.Bind(req, &s); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if len(s.Files) != len(files) {
		t.Fatalf("len=%d want %d", len(s.Files), len(files))
	}
	for i, f := range files {
		if s.Files[i].Filename != f.Filename {
			t.Fatalf("idx %d filename=%q want %q", i, s.Files[i].Filename, f.Filename)
		}
	}
}

func TestBBMultipartSliceRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 6))
	for i := 0; i < bbCases; i++ {
		n := rng.Intn(4) + 1
		files := make([]bbMultipartFile, n)
		for j := range files {
			files[j] = bbMultipartFile{"file", fmt.Sprintf("%d-%d.txt", i, j), []byte{byte(j)}}
		}
		req := bbCreateMultipartRequest(files, nil)
		var s struct {
			Files []multipart.FileHeader `form:"file"`
		}
		if err := binding.FormMultipart.Bind(req, &s); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if len(s.Files) != n {
			t.Fatalf("case %d: len=%d want %d", i, len(s.Files), n)
		}
	}
}

func TestBBMultipartArrayExactLen(t *testing.T) {
	files := []bbMultipartFile{
		{"file", "x1.txt", []byte("1")},
		{"file", "x2.txt", []byte("2")},
	}
	req := bbCreateMultipartRequest(files, nil)

	var s struct {
		Files [2]*multipart.FileHeader `form:"file"`
	}
	if err := binding.FormMultipart.Bind(req, &s); err != nil {
		t.Fatalf("bind: %v", err)
	}
	for i, f := range files {
		if s.Files[i].Filename != f.Filename {
			t.Fatalf("idx %d: %q != %q", i, s.Files[i].Filename, f.Filename)
		}
	}
}

func TestBBMultipartArrayLenInvalid(t *testing.T) {
	files := []bbMultipartFile{
		{"file", "a.txt", []byte("a")},
		{"file", "b.txt", []byte("b")},
	}
	req := bbCreateMultipartRequest(files, nil)

	var s struct {
		Files [1]*multipart.FileHeader `form:"file"`
	}
	err := binding.FormMultipart.Bind(req, &s)
	if !errors.Is(err, binding.ErrMultiFileHeaderLenInvalid) {
		t.Fatalf("expected ErrMultiFileHeaderLenInvalid, got %v", err)
	}
}

func TestBBMultipartWrongTypeError(t *testing.T) {
	files := []bbMultipartFile{{"file", "f.txt", []byte("x")}}
	req := bbCreateMultipartRequest(files, nil)

	var s struct {
		Files int `form:"file"`
	}
	err := binding.FormMultipart.Bind(req, &s)
	if !errors.Is(err, binding.ErrMultiFileHeader) {
		t.Fatalf("expected ErrMultiFileHeader, got %v", err)
	}
}

func TestBBMultipartWrongTypeRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 7))
	files := []bbMultipartFile{{"file", "f.txt", []byte("data")}}
	req := bbCreateMultipartRequest(files, nil)
	for i := 0; i < bbCases; i++ {
		switch rng.Intn(2) {
		case 0:
			var s struct {
				Files []int `form:"file"`
			}
			if err := binding.FormMultipart.Bind(req, &s); !errors.Is(err, binding.ErrMultiFileHeader) {
				t.Fatalf("case %d: []int under file key: %v", i, err)
			}
		case 1:
			var s struct {
				Files string `form:"file"`
			}
			if err := binding.FormMultipart.Bind(req, &s); !errors.Is(err, binding.ErrMultiFileHeader) {
				t.Fatalf("case %d: string under file key: %v", i, err)
			}
		}
	}
}

func TestBBMultipartFormValueFallback(t *testing.T) {
	req := bbCreateMultipartRequest(nil, map[string]string{"foo": "bar", "num": "42"})

	var s struct {
		Foo string `form:"foo"`
		Num int    `form:"num"`
	}
	if err := binding.FormMultipart.Bind(req, &s); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if s.Foo != "bar" || s.Num != 42 {
		t.Fatalf("got foo=%q num=%d", s.Foo, s.Num)
	}
}
