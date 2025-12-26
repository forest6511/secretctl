---
title: Contributing
description: Contribute to secretctl.
sidebar_position: 1
---

# Contributing to secretctl

Thank you for your interest in contributing to secretctl! This project is open source and welcomes contributions of all kinds.

## Ways to Contribute

### Report Bugs

Found a bug? Please [open an issue](https://github.com/forest6511/secretctl/issues/new) with:
- A clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Your environment (OS, Go version, secretctl version)

### Suggest Features

Have an idea for a new feature? Open an issue describing:
- The use case you're trying to solve
- How the feature would work
- Any alternatives you've considered

### Submit Code

Ready to contribute code? See the [Development Setup](/docs/contributing/development-setup) guide to get started.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Make your changes with tests
4. Run `go test ./...` and `golangci-lint run`
5. Submit a pull request

### Improve Documentation

Documentation improvements are always welcome:
- Fix typos or clarify confusing sections
- Add examples or use cases
- Translate documentation

## Code of Conduct

- Be respectful and constructive
- Focus on the issue, not the person
- Help others learn and grow

## Getting Started

- [Development Setup](/docs/contributing/development-setup) - Set up your development environment
- [Architecture Overview](/docs/architecture) - Understand the system design
- [GitHub Issues](https://github.com/forest6511/secretctl/issues) - Find issues to work on

## Pull Request Guidelines

### Before Submitting

- Run all tests: `go test ./...`
- Run linter: `golangci-lint run`
- Update documentation if needed
- Add tests for new functionality

### PR Description

Include:
- What the change does
- Why it's needed
- How to test it
- Related issue numbers

### Review Process

1. Maintainers will review your PR
2. Address any feedback
3. Once approved, maintainers will merge

## License

By contributing, you agree that your contributions will be licensed under the [Apache 2.0 License](https://github.com/forest6511/secretctl/blob/main/LICENSE).
