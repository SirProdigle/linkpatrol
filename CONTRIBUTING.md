# 🤝 Contributing to LinkPatrol

Thank you for your interest in contributing to LinkPatrol! This document provides guidelines and information for contributors.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
- [Development Setup](#development-setup)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Release Process](#release-process)

## 📜 Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code.

### Our Standards

- Use welcoming and inclusive language
- Be respectful of differing viewpoints and experiences
- Gracefully accept constructive criticism
- Focus on what is best for the community
- Show empathy towards other community members

## 🎯 How Can I Contribute?

### Reporting Bugs

- Use the GitHub issue tracker
- Include a clear and descriptive title
- Provide detailed steps to reproduce the issue
- Include system information (OS, Go version, etc.)
- Add error messages and logs if applicable

### Suggesting Enhancements

- Use the GitHub issue tracker with the "enhancement" label
- Describe the feature and its benefits
- Consider implementation complexity
- Provide use cases and examples

### Code Contributions

- Fork the repository
- Create a feature branch
- Make your changes
- Add tests
- Submit a pull request

## 🛠️ Development Setup

### Prerequisites

- Go 1.24.5 or higher
- Git
- Make (optional, for build scripts)

### Local Development

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/linkpatrol.git
cd linkpatrol

# Add upstream remote
git remote add upstream https://github.com/sirprodigle/linkpatrol.git

# Install dependencies
go mod download

# Build the project
go build -o linkpatrol

# Run tests
go test ./...
```

### Development Tools

We recommend using these tools for development:

- **gofmt**: Code formatting
- **golint**: Code linting
- **go vet**: Code analysis
- **golangci-lint**: Comprehensive linting

Install development tools:

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install additional tools
go install golang.org/x/tools/cmd/goimports@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
```

## 📝 Coding Standards

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Keep functions small and focused
- Use meaningful variable and function names
- Add comments for exported functions and types

### Code Organization

```
internal/
├── app/          # Main application logic
├── cache/        # Caching functionality
├── config/       # Configuration management
├── logger/       # Logging utilities
├── scanner/      # File scanning
├── tester/       # Link testing
├── walker/       # File parsing
├── watcher/      # File system watching
└── workers/      # Worker pool management
```

### Error Handling

- Always check errors
- Use `fmt.Errorf` with `%w` for wrapping errors
- Provide meaningful error messages
- Log errors appropriately

### Concurrency

- Use channels for communication between goroutines
- Implement proper context cancellation
- Avoid goroutine leaks
- Use sync.WaitGroup for coordination

## 🧪 Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run specific package tests
go test ./internal/app/...

# Run benchmarks
go test -bench=. ./...
```

### Writing Tests

- Write tests for all new functionality
- Use table-driven tests where appropriate
- Mock external dependencies
- Test both success and error cases
- Aim for >80% code coverage

### Test Structure

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "result",
            wantErr:  false,
        },
        // Add more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("FunctionName() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if result != tt.expected {
                t.Errorf("FunctionName() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## 🔄 Pull Request Process

### Before Submitting

1. **Update your branch**: `git pull upstream main`
2. **Run tests**: `go test ./...`
3. **Check formatting**: `gofmt -s -w .`
4. **Run linter**: `golangci-lint run`
5. **Update documentation** if needed

### Pull Request Guidelines

- Use a clear and descriptive title
- Provide a detailed description of changes
- Include any relevant issue numbers
- Add tests for new functionality
- Update documentation if needed
- Ensure all CI checks pass

### Commit Message Format

Use conventional commit format:

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test changes
- `chore`: Build/tooling changes

Examples:
```
feat(scanner): add support for YAML files
fix(cache): resolve memory leak in concurrent access
docs(readme): update installation instructions
```

## 🚀 Release Process

### Versioning

We use [Semantic Versioning](https://semver.org/) (SemVer):

- **MAJOR**: Incompatible API changes
- **MINOR**: New functionality (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Steps

1. **Create release branch**: `git checkout -b release/v1.2.3`
2. **Update version**: Update version in relevant files
3. **Update changelog**: Document changes
4. **Run full test suite**: `go test ./...`
5. **Create pull request**: Merge to main
6. **Tag release**: `git tag v1.2.3`
7. **Push tags**: `git push origin v1.2.3`
8. **Create GitHub release**: Add release notes

### Changelog

Maintain a `CHANGELOG.md` file with:

- New features
- Bug fixes
- Breaking changes
- Deprecations

## 📞 Getting Help

- **Issues**: [GitHub Issues](https://github.com/sirprodigle/linkpatrol/issues)
- **Discussions**: [GitHub Discussions](https://github.com/sirprodigle/linkpatrol/discussions)
- **Documentation**: [README.md](README.md)

## 🙏 Recognition

Contributors will be recognized in:

- Project README
- Release notes
- GitHub contributors page

Thank you for contributing to LinkPatrol! 🎉 