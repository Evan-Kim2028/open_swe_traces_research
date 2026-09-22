# Closure — ratehdrs

Package: github (root). File: github/github.go.
Removed bodies: parseRate (stub sets Reset even when the header parses to
0) and parseTokenExpiration (stub returns zero Timestamp).
Kept: Rate/Timestamp types, header name constants, Response wiring.
Tests removed: TestParseTokenExpiration in github/github_test.go.
