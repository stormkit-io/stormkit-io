# Contributing to Stormkit

Thank you for your interest in contributing to Stormkit! We welcome contributions from the community and are excited to see what you'll bring to the project.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
- [Development Setup](#development-setup)
- [Submitting Changes](#submitting-changes)
- [Coding Standards](#coding-standards)
- [License](#license)

## Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

### Community Edition vs Enterprise Edition

- **Community Edition (`src/ce/`)**: Licensed under AGPL-3.0, open for contributions
- **Enterprise Edition (`src/ee/`)**: Proprietary, contributions require signed CLA
- **Shared Libraries (`src/lib/`)**: Part of Community Edition, open for contributions

### Ways to Contribute

- 🐛 **Bug Reports**: Found a bug? Let us know!
- 💡 **Feature Requests**: Have an idea? We'd love to hear it!
- 🛠️ **Code Contributions**: Fix bugs, add features, improve performance
- 📖 **Documentation**: Help improve our docs
- 🧪 **Testing**: Write tests, test new features
- 🌍 **Translations**: Help us support more languages

## How to Contribute

### Reporting Issues

Before creating an issue, please:

1. **Search existing issues** to avoid duplicates
2. **Use the issue templates** when available
3. **Provide detailed information**:
   - Stormkit version
   - Operating system
   - Browser (if applicable)
   - Steps to reproduce
   - Expected vs actual behavior
   - Screenshots/logs if helpful

### Suggesting Features

We love feature suggestions! Please:

1. **Check existing feature requests** first
2. **Describe the problem** you're trying to solve
3. **Explain your proposed solution**
4. **Consider the scope**: CE vs EE features
5. **Think about implementation complexity**

## Development Setup

### Prerequisites

- **Go 1.25+**
- **Node.js 24+**
- **PostgreSQL 14+**
- **Redis 6+**
- **Docker**

### Local Development

See [Running the services](./README.md#running-the-services) in the main README for detailed setup instructions.

### Project Structure

```
stormkit-io/
├── src/
│   ├── ce/          # Community Edition (AGPL-3.0)
│   ├── ee/          # Enterprise Edition (Proprietary)
│   ├── lib/         # Shared libraries
│   └── migrations/  # Database migrations
├── scripts/         # Build and deployment scripts
└── docs/            # Documentation
```

## Submitting Changes

### Pull Request Process

1. **Fork the repository**
2. **Create a feature branch**

   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**

   - Follow our coding standards
   - Add tests for new functionality
   - Update documentation if needed

4. **Test your changes**

   See [Testing](../README.md#testing) section in the main README for more details.

5. **Commit your changes**

   ```bash
   git commit -m "feat: add new feature description"
   ```

   Use [Conventional Commits](https://www.conventionalcommits.org/) format:

   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation changes
   - `test:` for test changes
   - `refactor:` for code refactoring
   - `chore:` for maintenance tasks

6. **Push and create PR**

   ```bash
   git push origin feature/your-feature-name
   ```

   Then create a pull request on GitHub.

### PR Requirements

- ✅ **Descriptive title and description**
- ✅ **Tests pass** (automated checks)
- ✅ **Code review approved** by maintainers
- ✅ **Documentation updated** (if applicable)
- ✅ **No merge conflicts**
- ✅ **Signed commits** (if required)

## Coding Standards

### Go Code

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting
- Use `golangci-lint` for linting
- Write meaningful variable and function names
- Add comments for exported functions and types
- Keep functions small and focused

### Testing

- Write unit tests for new functions
- Use table-driven tests when appropriate
- Mock external dependencies
- Aim for good test coverage
- Test both success and error paths

### Documentation

- Use clear, concise language
- Include code examples where helpful
- Update README files when adding features
- Document breaking changes clearly

## License

### License Compliance

- **Community Edition**: Your contributions will be licensed under AGPL-3.0
- **Enterprise Edition**: Additional licensing terms may apply
- **Shared Libraries**: Licensed under AGPL-3.0 as part of CE

By contributing to this project, you agree that your contributions will be licensed under the same license as the component you're contributing to.

## Getting Help

### Community Support

- 💬 **Discussions**: Use GitHub Discussions for questions
- 🐛 **Issues**: Report bugs via GitHub Issues
- 📧 **Email**: Contact us at hello@stormkit.io
- 🌐 **Website**: Visit https://www.stormkit.io for more info

## Recognition

Contributors who make significant contributions may be:

- Listed in our CONTRIBUTORS file
- Mentioned in release notes
- Invited to join our contributor program
- Offered swag and recognition

---

Thank you for contributing to Stormkit! 🚀

_This document is adapted from open source contribution guidelines and follows best practices from the community._
