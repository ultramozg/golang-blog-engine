# Security Policy

## Supported Versions

We actively support the following versions with security updates:

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| develop | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it by:

1. **Do not** open a public GitHub issue
2. Email the maintainers directly with details
3. Include steps to reproduce the vulnerability
4. Provide any relevant code snippets or proof of concept

### What to expect

- Acknowledgment of your report within 48 hours
- Regular updates on the progress of fixing the vulnerability
- Credit for responsible disclosure (if desired)

## Security Measures

This project implements the following security measures:

- Automated dependency vulnerability scanning via GitHub Actions
- Static code analysis with gosec
- Regular dependency updates via Dependabot
- Secure coding practices and code review requirements

## Security Best Practices

When contributing to this project:

- Keep dependencies up to date
- Follow secure coding practices
- Validate all user inputs
- Use parameterized queries for database operations
- Implement proper authentication and authorization
- Handle errors securely without exposing sensitive information