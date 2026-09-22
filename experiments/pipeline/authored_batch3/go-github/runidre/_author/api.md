# Exported API — runidre

`DeploymentProtectionRuleEvent.GetRunID` pulls the workflow run ID out of
the event's `DeploymentCallbackURL`, so webhook handlers can correlate a
protection-rule callback with the run that triggered it.

- `DeploymentProtectionRuleEvent.GetRunID` — returns the numeric run ID or
  an error when the URL does not match the deployment-protection shape.
- `runIDFromURLRE` (unexported, kept) — the pattern the URL is matched
  against.
