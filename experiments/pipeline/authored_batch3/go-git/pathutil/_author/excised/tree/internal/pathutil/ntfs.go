package pathutil

import _ "strings"

// IsNTFSDotGit ports upstream Git's is_ntfs_dotgit. It detects path
// components that NTFS would resolve to ".git": the canonical name
// itself and its 8.3 short-name alias "git~1", each followed by any
// number of trailing spaces or periods (which NTFS silently trims)
// and an optional Alternate Data Stream suffix (":<stream>"). The
// bare strings ".git" and "git~1" also match, mirroring upstream.
//
// Reference: upstream Git path.c is_ntfs_dotgit at L1415-L1449
// in tag v2.54.0[1].
//
// [1]: https://github.com/git/git/blob/v2.54.0/path.c#L1415-L1449
func IsNTFSDotGit(part string) bool {
	var i int
	switch {
	case len(part) >= 4 && part[0] == '.' &&
		asciiToLower(part[1]) == 'g' &&
		asciiToLower(part[2]) == 'i' &&
		asciiToLower(part[3]) == 't':
		i = 4
	case len(part) >= 5 &&
		asciiToLower(part[0]) == 'g' &&
		asciiToLower(part[1]) == 'i' &&
		asciiToLower(part[2]) == 't' &&
		part[3] == '~' && part[4] == '1':
		i = 5
	default:
		return false
	}

	for ; i < len(part); i++ {
		c := part[i]
		if c == ':' {
			return true
		}
		if c != '.' && c != ' ' {
			return false
		}
	}
	return true
}

// WindowsValidPath reports whether part is a valid Windows / NTFS
// path component for the worktree filesystem abstraction. It rejects
// NTFS-disguised variants of `.git` and `git~1` (trailing spaces,
// periods, Alternate Data Streams) and Windows reserved device
// names. Bare `.git` and `git~1` are allowed at this layer; the
// caller decides whether they are permissible at the current path
// position.
func WindowsValidPath(part string) bool {
	panic("excised: WindowsValidPath")
}

// windowsReservedNames lists the Windows reserved device names.
// A path component is reserved if its base name (ignoring trailing
// spaces, extensions, and NTFS Alternate Data Streams) matches one of
// these case-insensitively.
//
// See upstream Git compat/mingw.c is_valid_win32_path().
var windowsReservedNames = []string{
	"CON", "PRN", "AUX", "NUL",
	"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
	"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	"CONIN$", "CONOUT$",
}

func isWindowsReservedName(part string) bool {
	panic("excised: isWindowsReservedName")
}

// IsNTFSDot ports upstream Git's is_ntfs_dot_generic. It detects NTFS
// path-component variants of a dotfile name that attackers can use to
// bypass case-insensitive comparisons against the canonical name on
// Windows. The dotgit parameter is the lowercase name without the
// leading dot (e.g. "gitmodules"); shortnamePrefix is the canonical
// 6-character NTFS short-name prefix used as a fall-back match
// (e.g. "gi7eba" for ".gitmodules").
//
// Reference: upstream Git path.c is_ntfs_dot_generic at L1451-L1507
// in tag v2.54.0[1].
//
// [1]: https://github.com/git/git/blob/v2.54.0/path.c#L1451-L1507
func IsNTFSDot(name, dotgit, shortnamePrefix string) bool {
	panic("excised: IsNTFSDot")
}

// IsNTFSDotGitmodules reports whether part is an NTFS equivalent of
// ".gitmodules" — the file name or any variant that NTFS would
// resolve to it. The 6-character canonical short-name prefix "gi7eba"
// mirrors upstream Git's is_ntfs_dotgitmodules.
func IsNTFSDotGitmodules(part string) bool {
	return IsNTFSDot(part, "gitmodules", "gi7eba")
}

// IsNTFSDotGitattributes reports whether part is an NTFS equivalent
// of ".gitattributes". The short-name prefix "gi7d29" mirrors upstream
// Git's is_ntfs_dotgitattributes.
func IsNTFSDotGitattributes(part string) bool {
	return IsNTFSDot(part, "gitattributes", "gi7d29")
}

// IsNTFSDotGitignore reports whether part is an NTFS equivalent of
// ".gitignore". The short-name prefix "gi250a" mirrors upstream Git's
// is_ntfs_dotgitignore.
func IsNTFSDotGitignore(part string) bool {
	return IsNTFSDot(part, "gitignore", "gi250a")
}

// IsNTFSDotMailmap reports whether part is an NTFS equivalent of
// ".mailmap". The short-name prefix "maba30" mirrors upstream Git's
// is_ntfs_dotmailmap.
func IsNTFSDotMailmap(part string) bool {
	return IsNTFSDot(part, "mailmap", "maba30")
}

func asciiToLower(c byte) byte {
	panic("excised: asciiToLower")
}
