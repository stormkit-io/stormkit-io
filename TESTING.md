# Test Coverage Guide

## Overview

This project aims for **80%+ code coverage** across both backend and frontend codebases. This guide explains how to work with coverage reports and improve coverage.

## Quick Start

### Run Tests with Coverage

```bash
# All tests
make test-coverage

# Backend only
make test-be

# Frontend only
make test-fe

# View coverage report
make coverage-report

# Generate HTML report
make coverage-html
```

## Coverage Thresholds

### Current Targets

| Component | Lines | Functions | Branches | Statements |
|-----------|-------|-----------|----------|------------|
| Backend   | 60%   | -         | -        | -          |
| Frontend  | 60%   | 60%       | 60%      | 60%        |

### Long-term Goals

| Component | Lines | Functions | Branches | Statements |
|-----------|-------|-----------|----------|------------|
| Backend   | 80%   | -         | -        | -          |
| Frontend  | 80%   | 80%       | 80%      | 80%        |

## Understanding Coverage Reports

### Backend (Go)

Coverage reports show:
- **Line coverage**: Percentage of code lines executed during tests
- **Coverage by package**: Breakdown by Go package

To view detailed coverage:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Frontend (React/TypeScript)

Coverage includes:
- **Lines**: Code lines executed
- **Statements**: JavaScript statements executed
- **Functions**: Functions called
- **Branches**: Conditional branches taken

To view detailed coverage:
```bash
cd src/ui
npm run test -- --coverage
open coverage/index.html  # macOS
xdg-open coverage/index.html  # Linux
start coverage/index.html  # Windows
```

## Best Practices

### 1. Test Critical Paths First

Focus on:
- Authentication & authorization
- Payment processing
- Data validation
- API endpoints
- Core business logic

### 2. Exclude Appropriate Files

Already excluded:
- Test files (`*_test.go`, `*.spec.tsx`)
- Mock files
- Type definitions
- Configuration files
- Generated code

### 3. Write Meaningful Tests

❌ **Bad**: Testing for 100% coverage without asserting behavior
```typescript
it('renders component', () => {
  render(<MyComponent />);
  // No assertions!
});
```

✅ **Good**: Testing actual behavior
```typescript
it('displays error message when form validation fails', () => {
  render(<MyComponent />);
  fireEvent.click(screen.getByText('Submit'));
  expect(screen.getByText('Email is required')).toBeInTheDocument();
});
```

### 4. Use Coverage Reports Wisely

Coverage metrics are **indicators**, not goals:
- 100% coverage ≠ bug-free code
- Focus on quality, not just quantity
- Use coverage to find untested code
- Prioritize high-risk areas

## CI/CD Integration

### Pull Request Checks

Coverage is checked on every PR:
1. Tests run automatically
2. Coverage report generated
3. Codecov comment added to PR
4. Threshold warnings shown if coverage drops

### Workflow Files

- Backend: `.github/workflows/test-be.yml`
- Frontend: `.github/workflows/test-fe.yml`

### Coverage Reports

Reports are uploaded to:
- **Codecov**: https://codecov.io/gh/stormkit-io/stormkit-io
- **GitHub Artifacts**: Available for 30 days

## Improving Coverage

### 1. Find Uncovered Code

```bash
# Backend: See uncovered lines
go tool cover -func=coverage.out | grep -v "100.0%"

# Frontend: Check HTML report
cd src/ui && npm run test -- --coverage
```

### 2. Common Patterns

#### Backend

```go
// ✅ Table-driven tests for comprehensive coverage
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "user@example.com", false},
        {"missing @", "userexample.com", true},
        {"empty string", "", true},
        {"multiple @", "user@@example.com", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

#### Frontend

```typescript
// ✅ Test user interactions and edge cases
describe('LoginForm', () => {
  it('handles successful login', async () => {
    const onSuccess = vi.fn();
    render(<LoginForm onSuccess={onSuccess} />);
    
    await userEvent.type(screen.getByLabelText('Email'), 'user@example.com');
    await userEvent.type(screen.getByLabelText('Password'), 'password123');
    await userEvent.click(screen.getByText('Login'));
    
    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalled();
    });
  });

  it('shows error on invalid credentials', async () => {
    render(<LoginForm />);
    
    await userEvent.click(screen.getByText('Login'));
    
    expect(screen.getByText('Invalid credentials')).toBeInTheDocument();
  });
});
```

### 3. Coverage Gaps Strategy

Priority order for increasing coverage:

1. **Critical security code** (auth, permissions)
2. **Data validation** (input sanitization)
3. **Core business logic** (deployments, billing)
4. **API endpoints** (REST handlers)
5. **Error handling** (edge cases)
6. **UI components** (user interactions)

## Troubleshooting

### Coverage Not Generated

```bash
# Backend: Ensure test database is running
docker compose up -d db redis

# Frontend: Clear cache
cd src/ui
rm -rf coverage/
npm run test -- --coverage
```

### Coverage Decreased

Check the Codecov comment on your PR for details:
- Which files lost coverage
- Which lines are now uncovered
- Suggestions for improvement

### Tests Timeout

```bash
# Backend: Increase timeout
go test -timeout 30m -coverprofile=coverage.out ./...

# Frontend: Check vitest.config.ts
# Adjust retry and timeout settings
```

## Resources

- [Go Testing Guide](https://golang.org/doc/tutorial/add-a-test)
- [Vitest Coverage](https://vitest.dev/guide/coverage.html)
- [Testing Library Best Practices](https://testing-library.com/docs/queries/about#priority)
- [Codecov Documentation](https://docs.codecov.com/)

## Questions?

For questions about test coverage:
1. Check existing tests in similar files
2. Ask in GitHub Discussions
3. Review this guide
4. Contact the team at hello@stormkit.io
