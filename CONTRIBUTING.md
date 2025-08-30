# Contributing to Klayengo

Thank you for your interest in contributing to Klayengo! We welcome contributions from the community.

## Code of Conduct

This project follows a code of conduct to ensure a welcoming environment for all contributors. By participating, you agree to:

- Be respectful and inclusive
- Focus on constructive feedback
- Accept responsibility for mistakes
- Show empathy towards other contributors
- Help create a positive community

## How to Contribute

### 1. Fork the Repository

Fork the repository on GitHub and clone your fork locally:

```bash
git clone https://github.com/your-username/klayengo.git
cd klayengo
git remote add upstream https://github.com/ambiyansyah-risyal/klayengo.git
```

### 2. Set Up Development Environment

```bash
# Install Go (version 1.21 or later)
# Install dependencies
go mod download

# Run tests to ensure everything works
go test ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

### 3. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-number-description
```

### 4. Make Your Changes

- Follow the existing code style
- Add tests for new functionality
- Update documentation as needed
- Ensure all tests pass
- Run benchmarks to check performance impact

### 5. Commit Your Changes

```bash
git add .
git commit -m "feat: add your feature description"
```

Use conventional commit format:
- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation
- `refactor:` for code refactoring
- `test:` for test additions
- `perf:` for performance improvements

### 6. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub using the provided template.

## Development Guidelines

### Code Style

- Follow standard Go formatting (`go fmt`)
- Use `gofmt` and `goimports`
- Follow Go naming conventions
- Keep functions focused and single-purpose
- Use meaningful variable and function names
- Add comments for complex logic

### Testing

- Write unit tests for all new functionality
- Aim for high test coverage (>90%)
- Use table-driven tests for multiple test cases
- Test error conditions and edge cases
- Run benchmarks for performance-critical code

### Documentation

- Update README.md for API changes
- Add examples for new features
- Document breaking changes
- Keep code comments up to date

### Performance

- Run benchmarks before and after changes
- Be mindful of memory allocations
- Consider concurrent performance
- Document performance implications

## Pull Request Process

1. **Fill out the PR template** completely
2. **Ensure CI passes** - all tests and checks must pass
3. **Get reviews** - at least one maintainer review required
4. **Address feedback** - make requested changes
5. **Merge** - maintainer will merge when ready

## Areas for Contribution

### High Priority
- Performance optimizations
- Bug fixes
- Security improvements
- Documentation improvements

### Medium Priority
- New features (circuit breaker algorithms, retry strategies)
- Additional middleware
- Integration with other monitoring systems

### Low Priority
- Additional examples
- Tooling improvements
- CI/CD enhancements

## Getting Help

- **Issues**: Use GitHub issues for bugs and feature requests
- **Discussions**: Use GitHub discussions for questions and ideas
- **Documentation**: Check README.md and examples first

## Recognition

Contributors will be:
- Listed in CHANGELOG.md
- Credited in release notes
- Added to a future contributors file

Thank you for contributing to Klayengo! 🎉
