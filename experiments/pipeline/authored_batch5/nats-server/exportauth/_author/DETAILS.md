# Details — exportauth

1. `checkStreamImportAuthorizedNoLock` returns false when the account has
   no stream exports OR when `subject` fails `IsValidSubject`; the service
   variant omits the validity check. Inferable: partially — asymmetry is
   deliberate but undocumented.
2. `checkStreamExportApproved` tries an exact map hit first: `ea == nil`
   means PUBLIC export → true; otherwise `checkAuth`. Inferable: yes —
   comment visible.
3. On a miss it iterates ALL export subjects and tests
   `isSubsetMatch(subjectTokens, exportSubj)` — the export may carry
   wildcards and the FIRST map-iteration match wins; the requesting
   subject must be a subset of (i.e. covered by) the export pattern.
   Inferable: partially — subset direction matters, comment notes import
   subject takes precedence.
4. `checkAuth` order: nil ea or (empty approved AND !tokenReq AND
   accountPos==0) → public true. Then `accountPos > 0` → authorized iff
   `accountPos <= len(tokens)` and `tokens[accountPos-1] == account.Name`
   (1-indexed position in the REQUESTING subject). Then `tokenReq` →
   `checkActivation`. Then `approved == nil` → false; else
   `approved[account.Name]` membership. Inferable: partially — accountPos
   semantics are only visible via tests.
5. `getServiceExport`/`getWildcardServiceExport`: exact map hit else
   iterate services with `isSubsetMatch` — same subset direction as #3.
   Inferable: partially.
6. `isRevoked`: empty map → false; revoked iff an entry for the subject
   OR the `jwt.All` wildcard exists with timestamp >= issuedAt; a stale
   (older) revocation does not count. Inferable: yes — jwt.All visible.
7. `checkUserRevoked` is a thin RLock wrapper over isRevoked on
   `a.usersRevoked`. Inferable: yes.
8. Locking contract: public `check*ImportAuthorized` take `a.mu.RLock`;
   the `NoLock` forms and `checkAuth`/`getServiceExport` require the
   caller to hold it. Inferable: doc — comments mark "Lock should be
   held".
