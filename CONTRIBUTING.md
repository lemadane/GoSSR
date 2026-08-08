# Contributing to GoSSR

Thank you for your interest in contributing to **GoSSR**! We welcome bug reports, feature proposals, performance improvements, and documentation enhancements.

---

## Development Setup

1. **Clone Repository**:
   ```bash
   git clone https://github.com/lemadane/gossr.git
   cd gossr
   ```

2. **Run Tests & Verification**:
   ```bash
   go test -v -race ./...
   ```

3. **Run Formatting & Linting**:
   ```bash
   gofmt -w .
   go vet ./...
   ```

4. **Run Benchmarks**:
   ```bash
   go test -bench=. -benchmem ./...
   ```

---

## Pull Request Guidelines

1. **Include Tests**: Every fix or feature must include unit tests in `engine_test.go`, `security_test.go`, or `custom_tags_test.go`.
2. **Preserve Security Contracts**: Any modification to `validateInterpolationSecurityContext` or `formatAndEscapeValueForContext` must maintain or enhance XSS protections.
3. **No Allocation Regressions**: Ensure changes do not introduce unnecessary memory allocations in the core rendering engine.
4. **Clean Code**: Ensure `gofmt -l .` reports zero unformatted files.
