# Contributing to secretctl

Thank you for your interest in contributing to secretctl! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for everyone.

## How to Contribute

### Reporting Bugs

Before submitting a bug report:

1. Check existing [issues](https://github.com/forest6511/secretctl/issues) to avoid duplicates
2. Use the latest version to confirm the bug still exists

When submitting a bug report, include:

- A clear, descriptive title
- Steps to reproduce the issue
- Expected vs actual behavior
- Your environment (OS, Go version, secretctl version)
- Relevant logs or error messages

### Suggesting Features

Feature requests are welcome! Please:

1. Check existing issues and discussions first
2. Clearly describe the problem the feature would solve
3. Provide use cases and examples

### Pull Requests

#### Branch Strategy

We use a simple branching model:

- `main` - stable, release-ready code
- `feature/*` - new features (e.g., `feature/add-export-command`)
- `fix/*` - bug fixes (e.g., `fix/unlock-timeout`)
- `docs/*` - documentation updates

#### Development Workflow

1. **Fork and clone** the repository

   ```bash
   git clone https://github.com/YOUR_USERNAME/secretctl.git
   cd secretctl
   ```

2. **Create a branch** from `main`

   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Set up the development environment**

   ```bash
   # Ensure Go 1.24+ is installed
   go version

   # Install dependencies
   go mod download

   # Verify the build
   go build ./...
   ```

4. **Make your changes**

   - Write clear, readable code
   - Follow existing code style and patterns
   - Add tests for new functionality
   - Update documentation as needed

5. **Run tests and linting**

   ```bash
   # Run all tests
   go test ./...

   # Run tests with race detector
   go test -race ./...

   # Run linter (requires golangci-lint)
   golangci-lint run ./...
   ```

6. **Commit your changes**

   Use [Conventional Commits](https://www.conventionalcommits.org/) format:

   ```
   feat: add password strength indicator
   fix: resolve unlock timeout on slow systems
   docs: update installation instructions
   test: add integration tests for export command
   refactor: simplify key derivation logic
   chore: update dependencies
   ```

7. **Push and create a Pull Request**

   ```bash
   git push origin feature/your-feature-name
   ```

   Then open a PR against `main` with:
   - A clear title and description
   - Reference to related issues (e.g., "Fixes #123")
   - Summary of changes made

#### PR Review Process

- All PRs require at least one review before merging
- CI checks must pass (tests, linting)
- Address review feedback promptly
- Keep PRs focused and reasonably sized

## Development Guidelines

### Code Style

- Follow standard Go conventions (`gofmt`, `goimports`)
- Use meaningful variable and function names
- Keep functions focused and under 40 lines when possible
- Add comments for non-obvious logic

### Testing

- Write table-driven tests where appropriate
- Aim for 80%+ test coverage on new code
- Include both positive and negative test cases
- Test edge cases and error conditions

### Security

This is a security-focused project. Please:

- Never log or expose sensitive data (passwords, keys, secrets)
- Use `crypto/rand` for all random number generation
- Follow secure coding practices
- Report security vulnerabilities privately (see SECURITY.md)

### Documentation

- Update README.md if adding user-facing features
- Add godoc comments for exported functions and types
- Keep documentation concise and accurate

## Getting Help

- 📚 Read the [Documentation](https://forest6511.github.io/secretctl/)
- Open a [discussion](https://github.com/forest6511/secretctl/discussions) for questions
- Check existing issues and documentation first
- Be patient and respectful when asking for help

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (Apache 2.0).

---

Thank you for contributing to secretctl!
