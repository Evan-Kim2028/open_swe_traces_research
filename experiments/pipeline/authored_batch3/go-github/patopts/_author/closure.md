# Closure — patopts

Package: github (root). File: github/orgs_personal_access_tokens.go.
Removed body: addListFineGrainedPATOptions — stubbed to delegate to
addOptions (which cannot emit the repeated array params).
Kept: ListFineGrainedPATOptions type, both list service methods,
PersonalAccessToken types.
Tests removed: 5 funcs in github/orgs_personal_access_tokens_test.go —
ListFineGrainedPersonalAccessTokens(_ownerOnly),
ListFineGrainedPersonalAccessTokenRequests(_ownerOnly/_tokenIDOnly),
TestOrganizationsService_ListFineGrainedPersonalAccessTokens.
