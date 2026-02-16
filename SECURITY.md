# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| latest  | Yes                |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public issue
2. Email the maintainers with details of the vulnerability
3. Include steps to reproduce the issue if possible
4. Allow reasonable time for a fix before public disclosure

We aim to acknowledge reports within 48 hours and provide a fix or mitigation plan within 7 days.

## Scope

Security concerns for this project include:
- Hash collision or integrity bypass in the content-addressed store
- Path traversal in pack creation or blob storage
- Arbitrary code execution during replay
- Information disclosure through pack metadata
