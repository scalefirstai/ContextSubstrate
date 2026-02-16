# Contributing to ContextSubstrate

Thank you for your interest in contributing! This guide covers how to report issues, suggest features, and submit code changes.

## Reporting Bugs

1. Check existing [issues](https://github.com/scalefirstai/ContextSubstrate/issues) to avoid duplicates
2. Use the **Bug Report** issue template
3. Include: version (`ctx --version`), OS, steps to reproduce, expected vs actual behavior
4. Attach relevant logs or error output

## Suggesting Features

1. Use the **Feature Request** issue template
2. Describe the problem you're trying to solve
3. Propose a solution and any alternatives you've considered

## Submitting Pull Requests

1. Fork the repository and create a feature branch from `main`
2. Make your changes following the guidelines below
3. Add or update tests as needed
4. Ensure all checks pass (`make test`, `make lint`, `make vet`)
5. Submit a PR using the pull request template

## Development Setup

### Prerequisites

- Go 1.25.6 or later
- [golangci-lint](https://golangci-lint.run/welcome/install/) (for linting)
- GNU Make

### Build

```bash
# Clone your fork
git clone https://github.com/<your-username>/ContextSubstrate.git
cd ContextSubstrate

# Build the binary
make build

# Run tests
make test

# Run linter
make lint

# Run all checks
make vet
make fmt
```

### Useful Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Compile `ctx` binary with version info |
| `make test` | Run all tests |
| `make coverage` | Run tests with coverage report |
| `make lint` | Run golangci-lint |
| `make vet` | Run `go vet` |
| `make fmt` | Check formatting |
| `make tidy` | Tidy go.mod |
| `make clean` | Remove build artifacts |
| `make install` | Install `ctx` to `$GOPATH/bin` |

## Code Style

- Follow standard Go conventions ([Effective Go](https://go.dev/doc/effective_go))
- Run `gofmt` before committing (or use `make fmt` to check)
- Keep functions focused and files organized by responsibility
- Write table-driven tests where applicable

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>: <description>

[optional body]
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `ci`, `chore`

Examples:
- `feat: add pack signing support`
- `fix: handle empty execution log gracefully`
- `docs: update command reference in README`
- `test: add integration tests for diff command`

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you agree to uphold this standard.
