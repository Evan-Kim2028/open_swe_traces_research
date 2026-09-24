package diff

import (
	"fmt"
	"io"
	_ "slices"
	"strconv"
	"strings"

	_ "example.internal/gitkit/v6/plumbing"
)

// DefaultContextLines is the default number of context lines.
const DefaultContextLines = 3

var (
	operationChar = map[Operation]byte{
		Add:    '+',
		Delete: '-',
		Equal:  ' ',
	}

	operationColorKey = map[Operation]ColorKey{
		Add:    New,
		Delete: Old,
		Equal:  Context,
	}
)

// UnifiedEncoder encodes an unified diff into the provided Writer. It does not
// support similarity index for renames or sorting hash representations.
type UnifiedEncoder struct {
	io.Writer

	// contextLines is the count of unchanged lines that will appear surrounding
	// a change.
	contextLines int

	// srcPrefix and dstPrefix are prepended to file paths when encoding a diff.
	srcPrefix string
	dstPrefix string

	// colorConfig is the color configuration. The default is no color.
	color ColorConfig
}

// NewUnifiedEncoder returns a new UnifiedEncoder that writes to w.
func NewUnifiedEncoder(w io.Writer, contextLines int) *UnifiedEncoder {
	return &UnifiedEncoder{
		Writer:       w,
		srcPrefix:    "a/",
		dstPrefix:    "b/",
		contextLines: contextLines,
	}
}

// SetColor sets e's color configuration and returns e.
func (e *UnifiedEncoder) SetColor(colorConfig ColorConfig) *UnifiedEncoder {
	e.color = colorConfig
	return e
}

// SetSrcPrefix sets e's srcPrefix and returns e.
func (e *UnifiedEncoder) SetSrcPrefix(prefix string) *UnifiedEncoder {
	e.srcPrefix = prefix
	return e
}

// SetDstPrefix sets e's dstPrefix and returns e.
func (e *UnifiedEncoder) SetDstPrefix(prefix string) *UnifiedEncoder {
	e.dstPrefix = prefix
	return e
}

// Encode encodes patch.
func (e *UnifiedEncoder) Encode(patch Patch) error {
	sb := &strings.Builder{}
	if message := patch.Message(); message != "" {
		sb.WriteString(message)
	}
	for _, filePatch := range patch.FilePatches() {
		e.writeFilePatchHeader(sb, filePatch)
		g := newHunksGenerator(filePatch.Chunks(), e.contextLines)
		for _, hunk := range g.Generate() {
			hunk.writeTo(sb, e.color)
		}
	}
	_, err := e.Write([]byte(sb.String()))
	return err
}

func (e *UnifiedEncoder) writeFilePatchHeader(sb *strings.Builder, filePatch FilePatch) {
	from, to := filePatch.Files()
	if from == nil && to == nil {
		return
	}
	var lines []string
	switch {
	case from != nil && to != nil:
		lines = append(lines, fmt.Sprintf("diff --git %s%s %s%s",
			e.srcPrefix, from.Path(), e.dstPrefix, to.Path()))
		lines = append(lines, fmt.Sprintf("index %s..%s", from.Hash(), to.Hash()))
		lines = e.appendPathLines(lines, e.srcPrefix+from.Path(), e.dstPrefix+to.Path(), false)
	case from == nil:
		lines = append(lines, fmt.Sprintf("diff --git %s%s %s%s",
			e.srcPrefix, to.Path(), e.dstPrefix, to.Path()))
		lines = append(lines, fmt.Sprintf("new file mode %o", to.Mode()))
		lines = e.appendPathLines(lines, "/dev/null", e.dstPrefix+to.Path(), false)
	case to == nil:
		lines = append(lines, fmt.Sprintf("diff --git %s%s %s%s",
			e.srcPrefix, from.Path(), e.dstPrefix, from.Path()))
		lines = append(lines, fmt.Sprintf("deleted file mode %o", from.Mode()))
		lines = e.appendPathLines(lines, e.srcPrefix+from.Path(), "/dev/null", false)
	}
	for _, line := range lines {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
}

func (e *UnifiedEncoder) appendPathLines(lines []string, fromPath, toPath string, isBinary bool) []string {
	return append(lines,
		fmt.Sprintf("--- %s", fromPath),
		fmt.Sprintf("+++ %s", toPath),
	)
}

type hunksGenerator struct {
	fromLine, toLine            int
	ctxLines                    int
	chunks                      []Chunk
	current                     *hunk
	hunks                       []*hunk
	beforeContext, afterContext []string
}

func newHunksGenerator(chunks []Chunk, ctxLines int) *hunksGenerator {
	return &hunksGenerator{
		chunks:   chunks,
		ctxLines: ctxLines,
	}
}

func (g *hunksGenerator) Generate() []*hunk {
	for _, chunk := range g.chunks {
		lines := splitLines(chunk.Content())
		switch chunk.Type() {
		case Equal:
			if g.current != nil {
				g.current.AddOp(Equal, lines...)
				g.hunks = append(g.hunks, g.current)
				g.current = nil
			}
			g.fromLine += len(lines)
			g.toLine += len(lines)
		case Delete:
			if g.current == nil {
				g.current = &hunk{fromLine: g.fromLine + 1, toLine: g.toLine}
			}
			g.current.AddOp(Delete, lines...)
			g.fromLine += len(lines)
		case Add:
			if g.current == nil {
				g.current = &hunk{fromLine: g.fromLine, toLine: g.toLine + 1}
			}
			g.current.AddOp(Add, lines...)
			g.toLine += len(lines)
		}
	}
	if g.current != nil {
		g.hunks = append(g.hunks, g.current)
	}
	return g.hunks
}

func (g *hunksGenerator) processHunk(i int, op Operation) {
}

// addLineNumbers obtains the line numbers in a new chunk.
func (g *hunksGenerator) addLineNumbers(la, lb, linesBefore, i int, op Operation) (cla, clb int) {
	return la, lb
}

func (g *hunksGenerator) processEqualsLines(ls []string, i int) {
}

func splitLines(s string) []string {
	return strings.SplitAfter(s, "\n")
}

type hunk struct {
	fromLine int
	toLine   int

	fromCount int
	toCount   int

	ctxPrefix string
	ops       []*op
}

func (h *hunk) writeTo(sb *strings.Builder, color ColorConfig) {
	sb.WriteString("@@ -")
	sb.WriteString(strconv.Itoa(h.fromLine))
	sb.WriteByte(',')
	sb.WriteString(strconv.Itoa(h.fromCount))
	sb.WriteString(" +")
	sb.WriteString(strconv.Itoa(h.toLine))
	sb.WriteByte(',')
	sb.WriteString(strconv.Itoa(h.toCount))
	sb.WriteString(" @@\n")
	for _, op := range h.ops {
		op.writeTo(sb, color)
	}
}

func (h *hunk) AddOp(t Operation, ss ...string) {
	n := len(ss)
	switch t {
	case Add:
		h.toCount += n
	case Delete:
		h.fromCount += n
	case Equal:
		h.toCount += n
		h.fromCount += n
	}
	for _, s := range ss {
		h.ops = append(h.ops, &op{s, t})
	}
}

type op struct {
	text string
	t    Operation
}

func (o *op) writeTo(sb *strings.Builder, color ColorConfig) {
	sb.WriteByte(operationChar[o.t])
	sb.WriteString(o.text)
	if !strings.HasSuffix(o.text, "\n") {
		sb.WriteByte('\n')
	}
}
