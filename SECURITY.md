# Security Policy

## Reporting Security Vulnerabilities

The GoSSR core team takes the security of our Server-Side Rendering engine seriously. If you discover a security vulnerability, HTML escaping bypass, or security scanner flaw, please report it responsibly rather than opening a public issue.

### How to Report a Vulnerability

Please email security reports to: **security@gossr.dev** (or open a private security advisory on GitHub).

Include the following information in your report:
- A clear description of the vulnerability or security bypass.
- Minimal reproducible Go example code demonstrating the vulnerability.
- Impact assessment (e.g. XSS, attribute breakout, context bypass).

### Disclosure & Response Process

1. **Acknowledgment**: We will acknowledge receipt of your report within 24 hours.
2. **Investigation**: We will investigate and confirm the issue within 48 hours.
3. **Patch & Release**: We will prepare a patch, run automated security unit tests and fuzzing suites, and publish a new version release.
4. **Public Advisory**: Credit will be given to the security researcher in the release notes and CHANGELOG.md unless requested otherwise.

---

## Security Model Overview

GoSSR uses a multi-layered security context parser:
- **Default Escaping**: `html.EscapeString` for all normal string interpolations.
- **URL Auto-Sanitization**: Protocol allowlist (`http`, `https`, `mailto`, `tel`, relative paths) for attributes like `href`, `src`, `action`, `formaction`, etc. (`javascript:` -> `about:blank`).
- **Context Rejection**: Automatic error rejection for `${...}` inside `<script>`, `<style>`, `<!-- -->`, `on*`, `style`, Alpine.js directives (`x-data`, `@click`), and HTMX executable attributes (`hx-on:*`).
- **Explicit Wrappers**: `gossr.Raw(...)` and `gossr.URL(...)` for explicitly trusted content.
