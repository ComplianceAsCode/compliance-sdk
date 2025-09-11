# Contributing to Compliance SDK

We welcome contributions to the Compliance SDK! This document provides guidelines for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Coding Standards](#coding-standards)
- [Documentation](#documentation)

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/). By participating, you are expected to uphold this code.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/your-username/compliance-sdk.git
   cd compliance-sdk
   ```
3. Add the upstream repository as a remote:
   ```bash
   git remote add upstream https://github.com/ComplianceAsCode/compliance-sdk.git
   ```

## Development Setup

### Prerequisites

- Go 1.24.0 or higher
- Make
- golangci-lint (for linting)
- Access to a Kubernetes cluster (for integration tests)

### Building the Project

```bash
# Install dependencies
go mod download

# Run tests
make test

# Run linter
make lint

# Format code
make fmt
```

## Making Changes

### Branch Naming

Use descriptive branch names:
- `feature/add-jsonpath-scanner` - for new features
- `fix/kubernetes-fetcher-panic` - for bug fixes
- `docs/update-api-examples` - for documentation updates
- `refactor/scanner-interface` - for refactoring

### Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
type(scope): subject

body

footer
```

Examples:
```
feat(scanner): add JSONPath rule engine support
fix(fetcher): handle nil pointer in Kubernetes fetcher
docs(api): update RuleBuilder examples
refactor(interfaces): simplify Input interface hierarchy
```

## Testing

### Unit Tests

- Write unit tests for all new functionality
- Maintain or improve code coverage
- Use table-driven tests where appropriate
- Mock external dependencies

Example test structure:
```go
func TestFeatureName(t *testing.T) {
    tests := []struct {
        name     string
        input    someType
        expected expectedType
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    validInput,
            expected: expectedOutput,
            wantErr:  false,
        },
        // more test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### Integration Tests

- Place integration tests in `*_integration_test.go` files
- Use build tags to separate integration tests: `// +build integration`
- Document any external dependencies required

### Running Tests

```bash
# Run unit tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests (requires Kubernetes cluster)
go test -tags=integration ./...
```

## Submitting Changes

### Pull Request Process

1. Update your fork with the latest upstream changes:
   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```

2. Create a new branch from main:
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. Make your changes and commit them

4. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

5. Create a Pull Request on GitHub

### Pull Request Guidelines

- Provide a clear description of the changes
- Reference any related issues
- Include tests for new functionality
- Update documentation as needed
- Ensure all tests pass
- Ensure code follows the project's style guidelines

### Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass (if applicable)
- [ ] Manual testing completed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] CHANGELOG.md updated (for significant changes)
```

## Coding Standards

### Go Style

- Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` for formatting
- Use `golangci-lint` for linting
- Keep functions focused and small
- Use meaningful variable and function names

### Interface Design

- Keep interfaces small and focused
- Document all exported types and functions
- Use interface segregation principle

### Error Handling

- Always handle errors explicitly
- Wrap errors with context using `fmt.Errorf`
- Use custom error types for specific error conditions
- Never ignore errors (use `_` only when absolutely necessary)

Example:
```go
if err != nil {
    return fmt.Errorf("failed to fetch resources: %w", err)
}
```

### Logging

- Use the provided Logger interface
- Log at appropriate levels (Debug, Info, Warn, Error)
- Include context in log messages
- Avoid logging sensitive information

## Documentation

### Code Documentation

- Document all exported types, functions, and methods
- Use complete sentences starting with the name being declared
- Include examples in documentation where helpful

Example:
```go
// RuleBuilder provides a fluent API for building compliance rules.
// It supports method chaining for easy rule construction.
type RuleBuilder struct {
    // ...
}

// WithInput adds an input source to the rule being built.
// The input will be available in the rule expression by its name.
func (b *RuleBuilder) WithInput(input Input) *RuleBuilder {
    // ...
}
```

### Documentation Updates

When adding new features or changing APIs:
1. Update the README.md if needed
2. Update API.md with new interfaces/types
3. Add examples to EXAMPLES.md
4. Update CHANGELOG.md

## Adding New Scanner Types

When implementing support for a new scanner type (e.g., Rego, JSONPath):

1. The `RuleType` constants are already defined for future use
2. Create implementation structs extending `BaseRule` (e.g., `JsonPathRuleImpl`)
3. Add builder methods (e.g., `SetJsonPathExpression()`)
4. Implement the scanner logic (e.g., `processJsonPathRule()`)
5. Update the switch statement in `Scanner.Scan()` and `RuleBuilder.Build()`
6. Add comprehensive tests
7. Update documentation

## Questions?

If you have questions about contributing, please:
1. Check existing issues and pull requests
2. Review the documentation
3. Open a new issue for discussion

Thank you for contributing to the Compliance SDK!
