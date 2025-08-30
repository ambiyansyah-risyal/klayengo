# Security Policy

## Supported Versions

We take security seriously and actively maintain the following versions with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in Klayengo, please help us by reporting it responsibly.

### How to Report

1. **Do not** create a public GitHub issue for the vulnerability
2. Email security concerns to: [security@klayengo.dev](mailto:security@klayengo.dev)
3. Include detailed information about:
   - The vulnerability description
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### What to Expect

- **Acknowledgment**: We'll acknowledge receipt within 48 hours
- **Investigation**: We'll investigate and provide regular updates
- **Fix**: We'll work on a fix and coordinate disclosure
- **Credit**: We'll credit you in the security advisory (if desired)

### Disclosure Policy

- We'll follow responsible disclosure practices
- We'll coordinate public disclosure with you
- We'll publish security advisories on GitHub
- We'll release patches before public disclosure

## Security Best Practices

When using Klayengo in production:

1. **Keep dependencies updated** - Regularly update to the latest version
2. **Use HTTPS** - Always use HTTPS for API communications
3. **Monitor logs** - Enable appropriate logging levels for security monitoring
4. **Rate limiting** - Configure appropriate rate limits for your use case
5. **Circuit breaker** - Use circuit breaker to prevent cascading failures
6. **Input validation** - Validate all inputs before processing

## Known Security Considerations

- **Memory usage**: Cache can grow unbounded - set appropriate size limits
- **Rate limiting**: Default settings may not be sufficient for high-traffic scenarios
- **Metrics exposure**: Prometheus metrics may expose sensitive information
- **Debug logging**: Debug logs may contain sensitive information

## Security Updates

Security updates will be:
- Released as patch versions (e.g., 1.2.3 → 1.2.4)
- Documented in release notes
- Marked with security advisories
- Backported to supported versions when applicable

## Contact

For security-related questions or concerns:
- Email: [security@klayengo.dev](mailto:security@klayengo.dev)
- GitHub Issues: For non-sensitive security questions only
