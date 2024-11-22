# Contributing to fstr

Thank you for your interest in contributing to fstr! This document provides guidelines and instructions for contributing.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/fstr.git`
3. Create a branch: `git checkout -b feature/your-feature-name`

## Development

### Prerequisites

- Go 1.19 or later
- golangci-lint (for linting)

### Running Tests

```bash
go test -v -race ./...
```

### Running Benchmarks

```bash
go test -bench=. -benchmem ./...
```

### Linting

```bash
golangci-lint run
```

## Pull Request Process

1. Update documentation if needed
2. Add tests for new features
3. Ensure all tests pass and linting is clean
4. Update CHANGELOG.md with your changes
5. Submit a pull request

## Code Style

- Follow standard Go formatting (gofmt)
- Add comments for exported functions and types
- Include examples in documentation
- Keep functions focused and small
- Write descriptive commit messages

## Reporting Issues

- Use the issue tracker
- Include Go version and OS
- Provide minimal reproduction steps
- Include error messages and stack traces

## License

By contributing, you agree that your contributions will be licensed under the MIT License. 