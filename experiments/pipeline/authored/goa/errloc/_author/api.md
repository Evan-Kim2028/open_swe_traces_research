# Exported API — errloc

```
func (m MultiError) Error() string
func (e *Error) Error() string
```

Error formats as `[file:line] message` when File is set, otherwise the wrapped Go error. MultiError joins those lines with newlines. Location helpers (unexported) walk the call stack, skip frames inside this module and registered design packages, keep test-file frames, strip `@version` path segments when matching packages, convert file paths to be relative to the working directory, and recover a source function's file:line via its func pointer for validation messages.

## Pre-existing callers

ReportError, ValidationErrors.Error, eval tests, any printer of evaluation errors.
