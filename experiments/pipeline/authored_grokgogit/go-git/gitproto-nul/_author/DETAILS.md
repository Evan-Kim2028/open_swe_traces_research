1. `Encode` refuses a nil writer with `ErrNilWriter`. Inferable: yes
2. An empty `RequestCommand` is invalid. Inferable: yes
3. Any ASCII control byte (0x00–0x1f or 0x7f) in command, pathname, host, or extra parameter is invalid. Inferable: no
4. The pkt-line payload is `command SP pathname NUL`, then `host=<host> NUL` when host is non-empty, then an extra NUL plus `param NUL` for each extra parameter when the extra-parameter list is non-empty. Inferable: no
5. The extra NUL before extra parameters is written even when host was also written, producing `...host=h NUL NUL param NUL`. Inferable: no
6. `Decode` of a flush packet or an empty line is `io.EOF`. Inferable: partially
7. The payload must end with a NUL. Inferable: no
8. The first space splits command from the rest; subsequent fields split on NUL. Inferable: partially
9. `Host` is the second field with a `host=` prefix stripped, not validated as actually starting with `host=`. Inferable: no
10. Empty extra-parameter fields are dropped. Inferable: no
11. Decode runs the same control-byte check as Encode. Inferable: yes
12. Invalid requests wrap `ErrInvalidGitProtoRequest`. Tests should match that sentinel, not a particular message. Inferable: no
