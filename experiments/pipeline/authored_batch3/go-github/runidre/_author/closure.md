# Closure — runidre

Package: github (root). File: github/github.go.
Removed body: DeploymentProtectionRuleEvent.GetRunID — stubbed to always
return (-1, "no match").
Kept: runIDFromURLRE regexp, DeploymentProtectionRuleEvent type.
Tests removed: TestDeploymentProtectionRuleEvent_GetRunID in
github/github_test.go.
