# Contributing

Thank you for considering a contribution to go-time-tracker!

## Reporting issues

Please open a GitHub issue and include:

- A clear description of the problem or feature request
- Steps to reproduce (for bugs)
- Your OS, terminal, and Go version

## Development setup

```bash
git clone https://github.com/kubeone/go-time-tracker.git
cd go-time-tracker
go mod download
go build -o gtt .
```

Run the tests:

```bash
go test ./...
```

## Submitting changes

1. Fork the repository and create a feature branch from `main`.
2. Keep changes focused — one logical change per pull request.
3. Make sure `go test ./...` and `go vet ./...` pass.
4. Write a clear commit message that explains the *why*, not just the *what*.
5. Open a pull request against `main` and describe what you changed and why.

## Storage format

`go-time-tracker` stores data as plain Markdown files in `~/.time-tracker/days/`.
Changes to the parsing or formatting logic must maintain backwards compatibility
with existing files so that users can continue to edit their records manually.

## Code style

- Follow standard Go conventions (`gofmt`, `go vet`).
- Avoid adding dependencies unless strictly necessary.
- Comments should explain non-obvious *why*, not restate the *what*.

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md).
By participating you agree to abide by its terms.
