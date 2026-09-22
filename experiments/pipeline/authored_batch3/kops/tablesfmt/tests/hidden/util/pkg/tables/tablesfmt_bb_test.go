// Package tables_test is the hidden black-box suite for tablesfmt.
// One TestDetailNN per DETAILS.md commitment. Exported API only:
// Table.AddColumn, Table.Render, SortByFunction.
package tables_test

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"example.internal/clustkit/util/pkg/tables"
)

type item struct {
	Name  string
	Value int
}

func newTable(t *testing.T) *tables.Table {
	t.Helper()
	tab := &tables.Table{}
	tab.AddColumn("name", func(i *item) string { return i.Name })
	tab.AddColumn("value", func(i *item) int { return i.Value })
	return tab
}

func renderLines(t *testing.T, tab *tables.Table, items []*item, cols ...string) []string {
	t.Helper()
	var buf bytes.Buffer
	if err := tab.Render(items, &buf, cols...); err != nil {
		t.Fatalf("Render: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	return lines
}

// Detail 1 (Inferable: yes): AddColumn stores the getter keyed by name; a
// duplicate name overwrites.
func TestDetail01(t *testing.T) {
	tab := &tables.Table{}
	tab.AddColumn("c", func(i *item) string { return "first" })
	tab.AddColumn("c", func(i *item) string { return "second" })
	lines := renderLines(t, tab, []*item{{Name: "n"}}, "c")
	if !strings.Contains(lines[1], "second") {
		t.Fatalf("duplicate AddColumn did not overwrite: %q", lines)
	}
}

// Detail 2 (Inferable: yes): the getter is invoked on the item and its FIRST
// return value is stringified.
func TestDetail02(t *testing.T) {
	tab := &tables.Table{}
	tab.AddColumn("s", func(i *item) string { return i.Name })
	tab.AddColumn("n", func(i *item) int { return i.Value })
	lines := renderLines(t, tab, []*item{{Name: "n1", Value: 42}}, "s", "n")
	if len(lines) != 2 || !strings.Contains(lines[1], "n1") || !strings.Contains(lines[1], "42") {
		t.Fatalf("getter values not rendered: %q", lines)
	}
}

// Detail 3 (Inferable: yes): SortByFunction adapts len/less/swap closures
// onto sort.Interface.
func TestDetail03(t *testing.T) {
	arr := []int{3, 1, 2}
	tables.SortByFunction(len(arr),
		func(i, j int) { arr[i], arr[j] = arr[j], arr[i] },
		func(i, j int) bool { return arr[i] < arr[j] })
	if arr[0] != 1 || arr[1] != 2 || arr[2] != 3 {
		t.Fatalf("not sorted ascending: %v", arr)
	}
	arr = []int{3, 1, 2}
	tables.SortByFunction(len(arr),
		func(i, j int) { arr[i], arr[j] = arr[j], arr[i] },
		func(i, j int) bool { return arr[i] > arr[j] })
	if arr[0] != 3 {
		t.Fatalf("not sorted descending: %v", arr)
	}
}

// Detail 4 (Inferable: partially): a missing column errors, mentioning the
// missing column name.
func TestDetail04(t *testing.T) {
	tab := newTable(t)
	var buf bytes.Buffer
	err := tab.Render([]*item{{Name: "x"}}, &buf, "nosuchcolumn")
	if err == nil {
		t.Fatal("missing column did not error")
	}
	if !strings.Contains(err.Error(), "nosuchcolumn") {
		t.Fatalf("error does not name the missing column: %v", err)
	}
}

// Detail 5 (Inferable: no): Render is fatal (process exit) when items is not
// a slice — not an error return.
func TestDetail05(t *testing.T) {
	if os.Getenv("BB_FATAL_CHILD") == "1" {
		tab := newTable(t)
		var buf bytes.Buffer
		_ = tab.Render(42, &buf, "name")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestDetail05")
	cmd.Env = append(os.Environ(), "BB_FATAL_CHILD=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("non-slice items did not abort the process: %s", out)
	}
}

// Detail 6 (Inferable: partially): rows sort lexicographically by column
// values, left-to-right, ties falling through to later columns.
func TestDetail06(t *testing.T) {
	tab := newTable(t)
	items := []*item{
		{Name: "b", Value: 9},
		{Name: "a", Value: 1},
		{Name: "b", Value: 3},
	}
	lines := renderLines(t, tab, items, "name", "value")
	if len(lines) != 4 {
		t.Fatalf("want header+3 rows: %q", lines)
	}
	// order: (a,1) (b,3) (b,9)
	var order []string
	for _, l := range lines[1:] {
		order = append(order, l)
	}
	if !strings.Contains(order[0], "a") {
		t.Fatalf("first row not 'a': %q", order)
	}
	if !(strings.Contains(order[1], "3") && strings.Contains(order[2], "9")) {
		t.Fatalf("tied rows not ordered by second column: %q", order)
	}
}

// Detail 7 (Inferable: no): every emitted cell is wrapped for tab-separation
// and the output is tabwriter-aligned. Asserted shape: a header line with
// every column name, one line per item with every cell value present.
func TestDetail07(t *testing.T) {
	tab := newTable(t)
	items := []*item{{Name: "n1", Value: 7}, {Name: "n2", Value: 8}}
	lines := renderLines(t, tab, items, "name", "value")
	if len(lines) != 3 {
		t.Fatalf("want header+2 rows: %q", lines)
	}
	for _, c := range []string{"name", "value"} {
		if !strings.Contains(lines[0], c) {
			t.Fatalf("header missing %q: %q", c, lines[0])
		}
	}
	for i, it := range items {
		if !strings.Contains(lines[i+1], it.Name) || !strings.Contains(lines[i+1], strconv.Itoa(it.Value)) {
			t.Fatalf("row %d missing cells: %q", i, lines[i+1])
		}
	}
}
