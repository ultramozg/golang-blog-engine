[![Test and Security](https://github.com/ultramozg/golang-blog-engine/actions/workflows/test.yml/badge.svg)](https://github.com/ultramozg/golang-blog-engine/actions)
[![codecov](https://codecov.io/gh/ultramozg/golang-blog-engine/branch/master/graph/badge.svg)](https://codecov.io/gh/ultramozg/golang-blog-engine)
[![Go Report Card](https://goreportcard.com/badge/github.com/ultramozg/golang-blog-engine)](https://goreportcard.com/report/github.com/ultramozg/golang-blog-engine)

This is the simple blog engine CMS which currently running on my Orange Pi PC.
Also i have rewritten all the part of my code to look more clear and readabble. old code can be seen on "old/main.go"
## CI/CD Pipeline

This project uses GitHub Actions for continuous integration and deployment. The pipeline includes:

### Automated Testing
- **Go 1.22 testing**: Tests run on the latest stable Go version
- **Race condition detection**: Tests run with `-race` flag
- **Code coverage**: Generates coverage reports and uploads to Codecov
- **Target coverage**: 80% minimum code coverage

### Security Scanning
- **Gosec**: Static security analysis for Go code
- **govulncheck**: Vulnerability scanning for dependencies
- **SARIF reporting**: Security findings uploaded to GitHub Security tab

### Code Quality
- **golangci-lint**: Comprehensive linting with multiple analyzers
- **Build verification**: Multi-platform build testing (Linux, macOS, Windows)
- **Dependency verification**: Ensures go.mod integrity

### Automated Updates
- **Dependabot**: Weekly dependency updates for Go modules and GitHub Actions
- **Security patches**: Automated security vulnerability fixes

### Branch Protection
To enable branch protection rules (recommended):

1. Go to repository Settings → Branches
2. Add rule for `master` branch:
   - Require status checks to pass before merging
   - Require branches to be up to date before merging
   - Select required status checks: `Test`, `Security Scan`, `Lint`, `Build`
   - Require pull request reviews before merging
   - Dismiss stale reviews when new commits are pushed

### Badges
Add these badges to display build status:

```markdown
[![Test and Security](https://github.com/ultramozg/golang-blog-engine/actions/workflows/test.yml/badge.svg)](https://github.com/ultramozg/golang-blog-engine/actions)
[![codecov](https://codecov.io/gh/ultramozg/golang-blog-engine/branch/master/graph/badge.svg)](https://codecov.io/gh/ultramozg/golang-blog-engine)
[![Go Report Card](https://goreportcard.com/badge/github.com/ultramozg/golang-blog-engine)](https://goreportcard.com/report/github.com/ultramozg/golang-blog-engine)
```

### Local Development
To run the same checks locally:

```bash
# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# Run security scan
gosec ./...

# Run linting
golangci-lint run

# Run vulnerability check
govulncheck ./...
```