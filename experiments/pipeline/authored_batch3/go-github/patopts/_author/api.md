# Exported API — patopts

`OrganizationsService.ListFineGrainedPersonalAccessTokens` and
`ListFineGrainedPersonalAccessTokenRequests` accept
`ListFineGrainedPATOptions`, whose Owner and TokenID filters must serialize
as repeated array query params the PAT endpoints expect.

- `addListFineGrainedPATOptions` (unexported, excised) — option→query
  serializer used by both list calls.
