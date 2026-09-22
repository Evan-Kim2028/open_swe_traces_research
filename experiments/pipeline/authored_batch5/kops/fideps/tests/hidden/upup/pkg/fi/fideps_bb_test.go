package fi

import (
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"
)

type bbLeaf struct {
	Name string
}

func (t *bbLeaf) Run(*Context[CloudupSubContext]) error { return nil }

// implements both Task and HasDependencies
type bbBoth struct {
	declared []Task[CloudupSubContext]
}

func (t *bbBoth) Run(*Context[CloudupSubContext]) error { return nil }
func (t *bbBoth) GetDependencies(tasks map[string]Task[CloudupSubContext]) []Task[CloudupSubContext] {
	return t.declared
}

// embeds NotADependency: reflective scan must be skipped
type bbShielded struct {
	NotADependency[CloudupSubContext]
	Dep *bbLeaf
}

func (t *bbShielded) Run(*Context[CloudupSubContext]) error { return nil }

// declares deps via HasDependencies; also has a task field
type bbDeclared struct {
	F        *bbLeaf
	declared []Task[CloudupSubContext]
}

func (t *bbDeclared) Run(*Context[CloudupSubContext]) error { return nil }
func (t *bbDeclared) GetDependencies(tasks map[string]Task[CloudupSubContext]) []Task[CloudupSubContext] {
	return t.declared
}

type bbHolder struct {
	Dep *bbLeaf
}

func (t *bbHolder) Run(*Context[CloudupSubContext]) error { return nil }

type bbContainers struct {
	S     string
	N     int
	Ptr   *bbLeaf
	Iface Task[CloudupSubContext]
	L     []*bbLeaf
	M     map[string]*bbLeaf
	MM    map[string]string
}

func (t *bbContainers) Run(*Context[CloudupSubContext]) error { return nil }

type bbResOnly struct{}

func (r *bbResOnly) Open() (io.Reader, error) { return strings.NewReader(""), nil }

type bbParentRes struct {
	R   *bbResOnly
	Dep *bbLeaf
}

func (t *bbParentRes) Run(*Context[CloudupSubContext]) error { return nil }

type bbPlain struct{ X string }

type bbParentPlain struct {
	P *bbPlain
}

func (t *bbParentPlain) Run(*Context[CloudupSubContext]) error { return nil }

type bbParentBoth struct {
	Sub *bbBoth
}

func (t *bbParentBoth) Run(*Context[CloudupSubContext]) error { return nil }

// non-task object for FindDependencies
type bbObj struct {
	Dep *bbLeaf
}

func bbSorted(xs []string) []string {
	out := append([]string(nil), xs...)
	sort.Strings(out)
	return out
}

func bbEqualSet(t *testing.T, got, want []string) {
	t.Helper()
	g, w := bbSorted(got), bbSorted(want)
	if len(g) != len(w) {
		t.Fatalf("deps = %v, want %v", g, w)
	}
	for i := range g {
		if g[i] != w[i] {
			t.Fatalf("deps = %v, want %v", g, w)
		}
	}
}

// TestDetail01: NotADependency.GetDependencies returns nil, and embedding it
// opts a task out of the reflective scan.
func TestDetail01(t *testing.T) {
	var n NotADependency[CloudupSubContext]
	if got := n.GetDependencies(nil); got != nil {
		t.Fatalf("NotADependency.GetDependencies = %v, want nil", got)
	}

	leaf := &bbLeaf{}
	shielded := &bbShielded{Dep: leaf}
	tasks := map[string]Task[CloudupSubContext]{"s": shielded, "l": leaf}
	deps := FindTaskDependencies(tasks)
	bbEqualSet(t, deps["s"], nil)
}

// TestDetail02: a task implementing HasDependencies reports its declared
// deps; reflection is not consulted.
func TestDetail02(t *testing.T) {
	leafA, leafB := &bbLeaf{}, &bbLeaf{}
	d := &bbDeclared{F: leafA, declared: []Task[CloudupSubContext]{leafB}}
	tasks := map[string]Task[CloudupSubContext]{"d": d, "a": leafA, "b": leafB}
	deps := FindTaskDependencies(tasks)
	bbEqualSet(t, deps["d"], []string{"b"})
}

// TestDetail03: dependencies are reported as task map keys.
func TestDetail03(t *testing.T) {
	leaf := &bbLeaf{}
	h := &bbHolder{Dep: leaf}
	tasks := map[string]Task[CloudupSubContext]{"holder": h, "leafkey": leaf}
	deps := FindTaskDependencies(tasks)
	bbEqualSet(t, deps["holder"], []string{"leafkey"})
	got, ok := deps["leafkey"]
	if !ok {
		t.Fatal("task with no deps has no map entry")
	}
	if len(got) != 0 {
		t.Fatalf("leaf task deps = %v, want empty", got)
	}
}

// TestDetail04: nil dependencies — typed or untyped — are skipped; a non-nil
// dep missing from the task map is fatal, not an error return (verified via
// a child process).
func TestDetail04(t *testing.T) {
	if os.Getenv("BB_FATAL_CHILD") == "1" {
		missing := &bbLeaf{}
		d := &bbDeclared{declared: []Task[CloudupSubContext]{missing}}
		FindTaskDependencies(map[string]Task[CloudupSubContext]{"d": d})
		os.Exit(0)
	}

	// nil deps skipped: untyped nil in the declared list
	leafB := &bbLeaf{}
	d := &bbDeclared{declared: []Task[CloudupSubContext]{nil, leafB}}
	tasks := map[string]Task[CloudupSubContext]{"d": d, "b": leafB}
	deps := FindTaskDependencies(tasks)
	bbEqualSet(t, deps["d"], []string{"b"})

	// typed nil skipped
	var nilTask *bbLeaf
	d2 := &bbDeclared{declared: []Task[CloudupSubContext]{nilTask}}
	deps = FindTaskDependencies(map[string]Task[CloudupSubContext]{"d": d2})
	bbEqualSet(t, deps["d"], nil)

	// nil task field skipped by reflection
	h := &bbHolder{Dep: nil}
	deps = FindTaskDependencies(map[string]Task[CloudupSubContext]{"h": h})
	bbEqualSet(t, deps["h"], nil)

	// missing dep is fatal
	cmd := exec.Command(os.Args[0], "-test.run=^TestDetail04$")
	cmd.Env = append(os.Environ(), "BB_FATAL_CHILD=1")
	if err := cmd.Run(); err == nil {
		t.Fatal("a dependency missing from the task map was silently accepted")
	}
}

// TestDetail05: the reflective walk ignores the root struct itself,
// primitives and strings, and descends through ptr/interface/slice/map.
func TestDetail05(t *testing.T) {
	l1, l2, l3, l4 := &bbLeaf{}, &bbLeaf{}, &bbLeaf{}, &bbLeaf{}
	c := &bbContainers{
		S:     "text",
		N:     42,
		MM:    map[string]string{"k": "v"},
		Ptr:   l1,
		Iface: l2,
		L:     []*bbLeaf{l3},
		M:     map[string]*bbLeaf{"x": l4},
	}
	tasks := map[string]Task[CloudupSubContext]{
		"c": c, "p": l1, "i": l2, "l": l3, "m": l4,
	}
	deps := FindTaskDependencies(tasks)
	bbEqualSet(t, deps["c"], []string{"p", "i", "l", "m"})
}

// TestDetail06: a struct field implementing both HasDependencies and Task
// contributes its declared deps AND itself; a Resource-only struct is
// ignored; any other non-task struct is fatal (verified via a child
// process).
func TestDetail06(t *testing.T) {
	if os.Getenv("BB_FATAL_CHILD") == "1" {
		p := &bbParentPlain{P: &bbPlain{X: "x"}}
		FindTaskDependencies(map[string]Task[CloudupSubContext]{"p": p})
		os.Exit(0)
	}

	leaf := &bbLeaf{}
	sub := &bbBoth{declared: []Task[CloudupSubContext]{leaf}}
	p := &bbParentBoth{Sub: sub}
	tasks := map[string]Task[CloudupSubContext]{"p": p, "sub": sub, "leaf": leaf}
	deps := FindTaskDependencies(tasks)
	bbEqualSet(t, deps["p"], []string{"sub", "leaf"})

	// Resource-only field ignored
	leaf2 := &bbLeaf{}
	pr := &bbParentRes{R: &bbResOnly{}, Dep: leaf2}
	tasks = map[string]Task[CloudupSubContext]{"pr": pr, "l": leaf2}
	deps = FindTaskDependencies(tasks)
	bbEqualSet(t, deps["pr"], []string{"l"})

	// a plain non-task struct field is an unhandled type — fatal
	cmd := exec.Command(os.Args[0], "-test.run=^TestDetail06$")
	cmd.Env = append(os.Environ(), "BB_FATAL_CHILD=1")
	if err := cmd.Run(); err == nil {
		t.Fatal("a non-task struct field was silently ignored")
	}
}

// TestDetail07: FindDependencies applies the same inference to an arbitrary
// non-task object.
func TestDetail07(t *testing.T) {
	leaf := &bbLeaf{}
	tasks := map[string]Task[CloudupSubContext]{"leaf": leaf}
	got := FindDependencies(tasks, &bbObj{Dep: leaf})
	if len(got) != 1 || got[0] != Task[CloudupSubContext](leaf) {
		t.Fatalf("FindDependencies = %v, want [leaf]", got)
	}
}
