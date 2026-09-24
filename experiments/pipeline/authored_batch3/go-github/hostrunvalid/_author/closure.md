# Closure — hostrunvalid

Package: github (root). File: github/actions_hosted_runners.go.
Removed body: validateCreateHostedRunnerRequest — stubbed to `return nil`.
Kept: CreateHostedRunnerRequest/UpdateHostedRunnerRequest types, both
CreateHostedRunner service methods.
Tests removed: TestActionsService_CreateHostedRunner in
github/actions_hosted_runners_test.go and
TestEnterpriseService_CreateHostedRunner in
github/enterprise_actions_hosted_runners_test.go.
