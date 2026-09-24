# DETAILS — ratehdrs

1. `parseRate` reads limit, remaining, used and resource from their headers
   when present. Inferable: yes — header names are the surviving constants.
2. `parseRate` leaves Reset at the zero timestamp when the reset header is
   absent or parses to 0. Inferable: no — treating a literal 0 epoch as
   "unset" is an arbitrary edge rule.
3. `parseTokenExpiration` accepts the `YYYY-MM-DD HH:MM:SS MST` layout and
   the `YYYY-MM-DD HH:MM:SS -0700` numeric-offset layout, returning the
   instant in local time. Inferable: partially — two accepted layouts are
   forced by observed server formats, the exact spellings are conventional.
4. Missing or unparseable expiration headers yield the zero Timestamp.
   Inferable: yes — documented in the comment above the function.
