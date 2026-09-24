# API left after excision

```go
func IsHFSDot(part, needle string) bool
func IsHFSDotGit(part string) bool
func IsHFSDotGitmodules(part string) bool
func IsHFSDotGitattributes(part string) bool
func IsHFSDotGitignore(part string) bool
func IsHFSDotMailmap(part string) bool
```

Wrappers pass a fixed lowercase needle: `"git"`, `"gitmodules"`, `"gitattributes"`, `"gitignore"`, `"mailmap"`.
