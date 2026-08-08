# Changelog

All notable changes to **GoSSR** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v0.1.0] - 2026-08-08

### Added
- **Pre-Compiled Template AST (`gossr.MustCompile`, `gossr.Compile`)**: Pre-validates and compiles template string ASTs for ultra-fast binding (`.Bind(scope)` and `.Render(writer, scope)`).
- **Strict Mode Validation (`gossr.Strict(true)`)**: Option to raise rendering errors on unresolved property placeholders and typos (`${properties.Custmer.Name}`).
- **Declarative JSX-Style Custom Component Tags (`gossr.Register`)**: Support for self-closing (`<Tag />`) and paired component tags (`<Tag>children</Tag>`).
- **Context-Aware Security Scanner**: Detects and rejects `${...}` interpolations inside `<script>`, `<style>`, `<!-- -->`, `on*`, `style`, Alpine.js directives (`x-data`, `@click`), and HTMX executable attributes (`hx-on:*`).
- **URL Context Protocol Auto-Sanitization**: Plain strings interpolated into URL attributes (`href`, `src`, `action`, etc.) with dangerous schemes (`javascript:`, `data:`) sanitize automatically to `about:blank`.
- **Explicit Security Wrappers**: `gossr.RawHtml` (`gossr.Raw(...)`) and `gossr.SafeUrl` (`gossr.URL(...)`).
- **Native HTTP Handlers**: `gossr.RenderHTTP(w, comp)` and `gossr.Handler(factoryFn)` with configurable `ErrorHandler` callback.
- **Attribute Security Boundary Enforcement**: `RawHtml` inside HTML attribute contexts (`attrContext != ""`) is automatically attribute-escaped to prevent attribute breakouts.
- **Nested Child Component Error Propagation**: Errors from child `SSR` component rendering propagate directly to parent components and `RenderHTTP` (returning HTTP 500) rather than being swallowed as HTML comments.
- **Panic-Proof Map Property Resolution**: Property path resolution safely handles `map[int]T`, `map[uint]T`, and non-string map key types via type checking and parsing instead of panicking in reflection.
- **Thread-Safe Strict Mode & Concurrency Protection**: Atomic `sync/atomic` implementation for `SetStrict(bool)` and `Strict(bool)` eliminating race conditions under concurrent requests.
- **Real Pre-Compiled AST Node Execution Graph**: `gossr.MustCompile` and `gossr.Compile` build an immutable node graph (`astStaticNode`, `astPropertyNode`, `astTernaryNode`, `astMapNode`), providing fast regex-free AST rendering.

### Performance
- Pre-compiled package regexes and AST pre-parsing, achieving **6.2 µs/render** (8.3x speedup, 91% reduction in memory allocations) and fast execution for large `.map()` slice iterations.
