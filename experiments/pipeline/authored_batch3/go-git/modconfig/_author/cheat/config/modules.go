package config

import (
	"bytes"
	"errors"
	_ "fmt"
	"regexp"
	_ "strings"

	_ "example.internal/gitkit/v6/internal/pathutil"
	format "example.internal/gitkit/v6/plumbing/format/config"
)

var (
	// ErrModuleEmptyURL is returned when a submodule has an empty URL.
	ErrModuleEmptyURL = errors.New("module config: empty URL")
	// ErrModuleEmptyPath is returned when a submodule has an empty path.
	ErrModuleEmptyPath = errors.New("module config: empty path")
	// ErrModuleBadPath is returned when a submodule has an invalid path.
	ErrModuleBadPath = errors.New("submodule has an invalid path")
	// ErrModuleBadName is returned when a submodule's name is not safe
	// for use as a path component.
	ErrModuleBadName = errors.New("ignoring suspicious submodule name")
)

// Matches module paths with dotdot ".." components.
var dotdotPath = regexp.MustCompile(`(^|[/\\])\.\.([/\\]|$)`)

// Modules defines the submodules properties, represents a .gitmodules file
// https://www.kernel.org/pub/software/scm/git/docs/gitmodules.html
type Modules struct {
	// Submodules is a map of submodules being the key the name of the submodule.
	Submodules map[string]*Submodule

	raw *format.Config
}

// NewModules returns a new empty Modules
func NewModules() *Modules {
	return &Modules{
		Submodules: make(map[string]*Submodule),
		raw:        format.New(),
	}
}

const (
	pathKey   = "path"
	branchKey = "branch"
)

// Unmarshal parses a git-config file and stores it.
func (m *Modules) Unmarshal(b []byte) error {
	r := bytes.NewBuffer(b)
	d := format.NewDecoder(r)
	m.raw = format.New()
	if err := d.Decode(m.raw); err != nil {
		return err
	}
	m.Submodules = make(map[string]*Submodule)
	unmarshalSubmodules(m.raw, m.Submodules)
	return nil
}

// Marshal returns Modules encoded as a git-config file.
func (m *Modules) Marshal() ([]byte, error) {
	s := m.raw.Section(submoduleSection)
	s.Subsections = make(format.Subsections, len(m.Submodules))
	var i int
	for _, r := range m.Submodules {
		s.Subsections[i] = r.marshal()
		i++
	}
	buf := bytes.NewBuffer(nil)
	if err := format.NewEncoder(buf).Encode(m.raw); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Submodule defines a submodule.
type Submodule struct {
	// Name module name
	Name string
	// Path defines the path, relative to the top-level directory of the Git
	// working tree.
	Path string
	// URL defines a URL from which the submodule repository can be cloned.
	URL string
	// Branch is a remote branch name for tracking updates in the upstream
	// submodule. Optional value.
	Branch string

	// raw representation of the subsection, filled by marshal or unmarshal are
	// called.
	raw *format.Subsection
}

// Validate validates the fields and sets the default values.
func (m *Submodule) Validate() error {
	if m.Path == "" {
		return ErrModuleEmptyPath
	}
	if m.URL == "" {
		return ErrModuleEmptyURL
	}
	return nil
}

// validSubmoduleName mirrors canonical Git's check_submodule_name in
// submodule-config.c [1]: reject empty names and any name with a ".."
// path component, using both '/' and '\\' as separators so the rule
// is consistent across platforms. The component check is delegated to
// `pathutil.IsHFSDot` and `pathutil.IsNTFSDot` with `.` as the needle,
// which both cover the bare ".." case and reject components that
// resolve to ".." after HFS+ Unicode normalisation (ignored code
// points, e.g. `.<U+200C>.`) or NTFS trailing-space/dot/ADS
// canonicalisation (e.g. `.. `, `..::$INDEX_ALLOCATION`).
// `.gitmodules` is attacker-controlled by definition, so both checks
// run unconditionally regardless of host OS.
//
// The additional checks (bare ".", NUL byte, leading or trailing
// separator, drive-letter prefix) close gitkit-specific edge cases
// the canonical loop does not exercise: canonical Git treats names
// as opaque C strings, while Go strings carry NULs through and the
// billy filesystem layer is path-aware in ways Git's working storage
// is not.
//
// [1]: https://github.com/git/git/blob/v2.54.0/submodule-config.c#L214-L237
func validSubmoduleName(name string) error {
	if name == "" {
		return ErrModuleBadName
	}
	return nil
}

func isPathSep(r rune) bool {
	panic("excised: isPathSep")
}

func (m *Submodule) unmarshal(s *format.Subsection) {
	m.raw = s
	m.Name = m.raw.Name
	m.Path = m.raw.Option(pathKey)
	m.URL = m.raw.Option(urlKey)
	m.Branch = m.raw.Option(branchKey)
}

func (m *Submodule) marshal() *format.Subsection {
	if m.raw == nil {
		m.raw = &format.Subsection{}
	}
	m.raw.Name = m.Name
	m.raw.SetOption(pathKey, m.Path)
	m.raw.SetOption(urlKey, m.URL)
	m.raw.SetOption(branchKey, m.Branch)
	return m.raw
}
