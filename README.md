[![Test and Security](https://github.com/ultramozg/golang-blog-engine/actions/workflows/test.yml/badge.svg)](https://github.com/ultramozg/golang-blog-engine/actions)
[![codecov](https://codecov.io/gh/ultramozg/golang-blog-engine/branch/master/graph/badge.svg)](https://codecov.io/gh/ultramozg/golang-blog-engine)
[![Go Report Card](https://goreportcard.com/badge/github.com/ultramozg/golang-blog-engine)](https://goreportcard.com/report/github.com/ultramozg/golang-blog-engine)

This is the simple blog engine CMS which currently running on my Orange Pi PC.
Also i have rewritten all the part of my code to look more clear and readabble. old code can be seen on "old/main.go"


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