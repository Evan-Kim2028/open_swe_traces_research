# Exported API — hostrunvalid

`ActionsService.CreateHostedRunner` and
`EnterpriseService.CreateHostedRunner` reject malformed requests client-side
before issuing the POST.

- `validateCreateHostedRunnerRequest` (unexported, excised) — required-field
  predicate shared by both call sites.
