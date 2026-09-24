# Closure — boolresp

Package: github (root). File: github/github.go.
Removed bodies: parseBoolResponse — stubbed to propagate every error
(drops the 404 → (false, nil) swallow).
Kept: ErrorResponse type, CheckResponse, all predicate callers.
Tests removed: TestParseBooleanResponse_{true,false,error} in
github/github_test.go, plus the 8 predicate tests that exercise the same
404-swallow through their services: TestUsersService_IsFollowing_false
(users_followers_test.go), TestRepositoriesService_IsCollaborator_False
(repos_collaborators_test.go), TestOrganizationsService_IsPublicMember_notMember
and TestOrganizationsService_IsMember_notMember (orgs_members_test.go),
TestIssuesService_IsAssignee_false (issues_assignees_test.go),
TestGistsService_IsStarred_noStar (gists_test.go),
TestActivityService_GetRepositorySubscription_false
(activity_watching_test.go), TestActivityService_IsStarred_noStar
(activity_star_test.go).
