# Contribution guidelines

Dubbo Kubernetes is released under the non-restrictive Apache 2.0 license and follows a very standard GitHub development process, using GitHub tracker for issues and merging pull requests into master. Contributions of all forms to this repository are acceptable, as long as they follow the prescribed community guidelines enumerated below.

## Before opening an issue

- Search existing issues before filing a new one.
- Use the bug report template for reproducible incorrect behavior and the
  feature request template for a concrete new capability or improvement.
- Include the smallest useful reproduction, version and environment details,
  and sanitized logs. Never post credentials, private data, or security
  vulnerabilities in a public issue.
- Report suspected security vulnerabilities through the private process at
  <https://www.apache.org/security/>.

## Before opening a pull request

- Keep each pull request focused on one problem and link the related issue.
- Describe compatibility and rollout implications for API, configuration,
  deployment, performance, or security changes.
- Run the relevant tests and static checks. `make verify` runs the checks that
  are fast enough for normal pre-push validation.
- Add or update tests for changed behavior and update user-facing documentation
  or examples when needed.
- Complete the pull request checklist and include exact verification evidence.

## Bot commands

Comment on an issue or pull request with one of the following commands (the command must start the comment):

| Command | Where | Who | Effect |
|---|---|---|---|
| `/assign [@user ...]` | issue / PR | anyone (self); write access (others) | Assign the issue or PR |
| `/unassign [@user ...]` | issue / PR | anyone (self); write access (others) | Remove assignees |
| `/lgtm` | PR | approvers in the root [`OWNERS`](OWNERS) file | Record maintainer approval and enable GitHub Auto-merge; the PR is squash-merged after required checks pass |
| `/lgtm cancel` | PR | write access or PR author | Remove the `lgtm` label and disable auto-merge |
| `/close` | issue / PR | author or write access | Close |
| `/reopen` | issue / PR | author or write access | Reopen |
| `/retest` | PR | author or write access | Re-run failed workflow runs for the PR head commit |

## Automatic pull request labels

The GitHub bot labels pull requests from their changed paths and branch names.
Component labels include `cni`, `dubboctl`, `dubbod`, `operator`, `helm`,
`networking`, and `samples`. Cross-cutting labels include `documentation`,
`testing`, `performance`, `security`, `build`, `release`, `dependencies`, and
`github_actions`.

Pushes to a pull request reset its maintainer approval and remove the `lgtm`
label. An approver must review the new head and comment `/lgtm` again.

Thank you for contributing to Dubbo Kubernetes!
