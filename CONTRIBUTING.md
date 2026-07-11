# Contribution guidelines

Dubbo Kubernetes is released under the non-restrictive Apache 2.0 license and follows a very standard GitHub development process, using GitHub tracker for issues and merging pull requests into master. Contributions of all forms to this repository are acceptable, as long as they follow the prescribed community guidelines enumerated below.

## Bot commands

Comment on an issue or pull request with one of the following commands (the command must start the comment):

| Command | Where | Who | Effect |
|---|---|---|---|
| `/assign [@user ...]` | issue / PR | anyone (self); write access (others) | Assign the issue or PR |
| `/unassign [@user ...]` | issue / PR | anyone (self); write access (others) | Remove assignees |
| `/lgtm` | PR | collaborators with write access (not the PR author) | Add the `lgtm` label; if the commenter is an approver in the root [`OWNERS`](OWNERS) file, the PR is squash-merged (auto-merge if checks are still running) |
| `/lgtm cancel` | PR | write access or PR author | Remove the `lgtm` label and disable auto-merge |
| `/close` | issue / PR | author or write access | Close |
| `/reopen` | issue / PR | author or write access | Reopen |
| `/retest` | PR | author or write access | Re-run failed workflow runs for the PR head commit |

Thank you for contributing to Dubbo Kubernetes!
