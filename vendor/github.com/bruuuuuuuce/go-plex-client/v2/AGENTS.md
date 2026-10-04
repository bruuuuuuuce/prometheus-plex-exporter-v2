# Repository Guidelines

## Workflow

- Do all work in a dedicated Git worktree and feature branch. Do not make changes in the primary checkout.
- Keep each branch focused on one change and use the `codex/` prefix for agent-created branches.
- Always submit changes through a pull request. Never commit or push directly to the default branch.
- Keep commits small, coherent, and free of unrelated formatting or cleanup.
- Always write commit messages using Conventional Commits: `<type>[optional scope]: <description>` (for example, `docs: clarify installation` or `fix(websocket): handle normal closure`). Use an imperative, lowercase description without a trailing period. Common types are `feat`, `fix`, `docs`, `test`, `refactor`, `ci`, and `chore`. Mark breaking changes with `!` and explain them in a `BREAKING CHANGE:` footer.
- Preserve existing user changes and do not use destructive Git commands unless explicitly requested.

## Development

- Follow test-driven development: add or update a failing test first, implement the smallest change that makes it pass, then refactor while the tests remain green.
- Match the existing Go style and APIs. Prefer simple, idiomatic code over new abstractions unless they clearly reduce complexity.
- Run `gofmt` on changed Go files.
- Add tests for bug fixes, new behavior, and meaningful edge cases. Avoid tests that depend on external Plex services when a deterministic local test is possible.

## Validation

- Run `go test ./...` before opening or updating a pull request.
- Run any additional checks introduced by the repository or CI configuration.
- Ensure CI passes before considering the work complete. If a check cannot be run locally, state that clearly in the pull request.
- Review the final diff for accidental changes, generated files, secrets, and debug output.

## Pull Requests

- Explain what changed, why it changed, and how it was tested.
- Link the relevant issue when one exists.
- Keep the pull request scoped and address failing CI before requesting review.
