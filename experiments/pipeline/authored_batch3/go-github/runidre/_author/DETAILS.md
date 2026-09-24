# DETAILS — runidre

1. A callback URL containing `repos/<org>/<repo>/actions/runs/<digits>/
   deployment_protection_rule` yields that digit run as an int64.
   Inferable: doc — the regexp constant survives and fixes the shape.
2. Both absolute and relative URLs satisfying the pattern work. Inferable:
   yes — the regex is not anchored to a scheme.
3. A URL that does not match the pattern returns -1 and an error.
   Inferable: yes — signature contract; the error literal is arbitrary.
4. A matched capture that fails integer parsing returns -1 and the parse
   error. Inferable: no — propagating the parse error vs a sentinel is an
   arbitrary choice.
