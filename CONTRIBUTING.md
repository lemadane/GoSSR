# Contributing to GoSSR

Thank you for your interest in contributing to **GoSSR**! We welcome bug reports, feature proposals, performance optimizations, and documentation enhancements.

---

## Development & Testing Workflow

1. **Clone Repository**:
   ```bash
   git clone https://github.com/lemadane/gossr.git
   cd gossr
   ```

2. **Run Tests & Data Race Verification**:
   ```bash
   go test -v -race ./...
   ```

3. **Run Code Formatting & Static Analysis**:
   ```bash
   gofmt -w .
   go vet ./...
   ```

4. **Run Benchmarks**:
   ```bash
   go test -bench=. -benchmem ./...
   ```

5. **Run Security & Template Fuzzing**:
   ```bash
   go test -fuzz=FuzzRender -fuzztime=30s ./...
   go test -fuzz=FuzzSanitizeUrl -fuzztime=30s ./...
   ```

---

## Pull Request Guidelines

1. **Include Unit Tests**: Every fix or feature must include unit tests in `engine_test.go`, `security_test.go`, `custom_tags_test.go`, `forms_test.go`, or `edge_cases_test.go`.
2. **AST Feature Parity**: Any new template syntax or expression feature MUST update both `Render()` and `Compile()` AST node parsers (`parseTemplateAST`). Output equality must be verified via `TestCompiledTemplateParity`.
3. **Preserve Security Boundaries**: Any modification to `validateInterpolationSecurityContext` or `formatAndRenderValueForContext` must maintain or enhance XSS protections, attribute quote escaping, and URL protocol auto-sanitization.
4. **No Allocation Regressions**: Ensure changes do not introduce unnecessary heap memory allocations in core AST node rendering loops (`renderAST`).
5. **Clean Formatting**: Ensure `gofmt -l .` reports zero unformatted files before submitting PRs.

---

## Bug Reports & Feature Requests

- Open a detailed issue on [GitHub Issues](https://github.com/lemadane/gossr/issues).
- Provide minimal reproducible Go code for bug reports.
- For security vulnerabilities, follow the reporting process described in [SECURITY.md](SECURITY.md).
