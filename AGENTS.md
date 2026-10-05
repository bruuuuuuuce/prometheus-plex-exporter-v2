# Repository guidance

This file applies to the entire repository. Keep changes small, reviewable, and tied to one issue or purpose.

## Workflow

- Do all work in a dedicated Git worktree and feature branch. Do not make
  changes in the primary checkout.
- Do not commit or push directly to `main`. Create a descriptive branch, then open a pull request for review.
- Always write commit messages using Conventional Commits:
  `<type>[optional scope]: <description>` (for example,
  `docs: clarify installation` or `fix(config): handle missing values`).
  Use an imperative, lowercase description without a trailing period. Common
  types are `feat`, `fix`, `docs`, `test`, `refactor`, `ci`, and `chore`.
  Mark breaking changes with `!` and explain them in a `BREAKING CHANGE:` footer.
- Check `git status` before editing. Preserve unrelated changes and avoid destructive Git commands.
- Describe the behavior changed, verification performed, and any compatibility or deployment impact in the pull request. Link the relevant issue.
- Review the pull request diff and wait for required checks before merging. Request another reviewer when available.

## Code and verification

- This is a Go 1.23 project. The application entry point is `cmd/prometheus-plex-exporter`; Plex integration is in `pkg/plex`, and metric definitions are in `pkg/metrics`.
- Keep Go code formatted with `gofmt`. Run `go test -mod=vendor ./...` and `go build -mod=vendor -o /dev/null ./cmd/prometheus-plex-exporter` for code changes. Run any narrower checks needed for the affected behavior.
- Add a regression test when fixing a bug or changing observable behavior. Tests should check outcomes, such as decoded events or emitted metric labels, rather than only mirror implementation details.
- Dependencies are vendored. Make dependency updates deliberately, including `go.mod`, `go.sum`, and `vendor/` together. Avoid unexplained edits inside `vendor/`.
- Preserve metric names, types, labels, and label meanings where possible. Treat corrections that change Prometheus series identity as compatibility changes and document them.
- Update `README.md` and examples when configuration, metrics, or deployment behavior changes.

## Security and operations

- Never commit Plex tokens, credentials, or private configuration. Do not log tokens or full Plex payloads containing user or media details.
- Keep diagnostic logs useful at normal operating levels; put high-volume expected events behind an appropriate lower level.
- Docker publishing is a separate release task. Do not publish an image as part of an unrelated change.
