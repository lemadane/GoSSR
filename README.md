# GoSSR

**GoSSR** (**Server-Side Rendered HTML**) is a lightweight, pure, zero-transpilation framework for Go that brings a React-like Developer Experience (DX) directly to native Go backtick string templates.

With **GoSSR**, you write component-based user interfaces in pure `.go` files with **zero external build steps, zero node_modules, zero transpilers, and zero code generators**.

---

## Key Features

- **Clean & Reusable Go Package**: Importable cleanly as `import "github.com/lemadane/gossr"`.
- **React-like DX in Pure Go**: Define reusable UI components using standard Go functions returning the `gossr.SSR` (Server-Side Rendered HTML) interface.
- **Declarative JSX-Style Custom Tags (`gossr.Register`)**: Register custom component tags (`gossr.Register("UserBadge", UserBadgeFunc)`) and compose self-closing (`<UserBadge name="Sarah" role="Admin" />`) or paired tags (`<Card title="Settings"><h1>Body Content</h1></Card>`) directly in HTML templates.
- **Strict Custom Tag Attribute Validation**: Automatically validates attributes passed to custom component tags against target struct fields and raises a clear error on typos (`<UserCard nme="Sarah" />`).
- **Zero CLI Build Steps**: Runs directly with standard `go run` or `go build` with zero node_modules or transpilers.
- **Comprehensive HTML/XSS Protection**: Standard string and property substitutions are automatically HTML-escaped (`html.EscapeString`). Plain strings interpolated into URL-bearing attributes (`href`, `src`, `action`, `formaction`, `cite`, `data`, `poster`, `icon`) are automatically sanitized (`SanitizeUrl`) to `about:blank`. `gossr.SSR` components and explicit `gossr.RawHtml` wrappers are preserved as trusted raw HTML.
- **Context-Aware HTML Security Scanner**: Enforces strict security rules by detecting and blocking dangerous `${...}` interpolations inside `<script>` blocks, `<style>` blocks, HTML comments (`<!-- ... -->`), inline event handler attributes (`onclick`, `onload`, `on*`), inline `style="..."` attributes, Alpine.js directives (`x-data`, `x-init`, `x-effect`, `x-on:*`, `@*`, `x-bind:*`, `: *`), and HTMX executable attributes (`hx-on:*`, `hx-vals`, `hx-headers`, `hx-vars`).
- **Explicit Security Wrappers**:
  - `gossr.RawHtml` (`gossr.Raw("<b>trusted</b>")`): Mark explicitly trusted HTML content to bypass escaping.
  - `gossr.SafeUrl` (`gossr.URL("https://...")`): Enforces a strict URL protocol allowlist (`http`, `https`, `mailto`, `tel`, relative paths `/`, `#`, `?`, `.`) while sanitizing dangerous protocols (`javascript:`, `vbscript:`, `data:`) to `about:blank`.
- **Template Expression Parsing**:
  - `${properties.FieldName}`: Top-level property evaluation.
  - `${properties.Parent.Child.Deep...}`: Recursive nested property path resolution to any depth.
  - `${properties.Children}`: Embedded child `gossr.SSR` component rendering.
  - `${properties.Condition ? "OptionA" : "OptionB"}`: Inline ternary conditional evaluation.
  - `${properties.Role == "ADMIN" ? "checked" : ""}`: Equality comparison ternaries for Checkboxes and Radio Buttons.
  - `${properties.Slice.map(item => <template>${item.Field}</template>)}`: Slice mapping with literal dollar sign (`$`) preservation and template variable evaluation.
- **Strict Mode Validation (`gossr.Strict(true)`)**: Enables development/strict mode where typos in property paths (`${properties.Custmer.Name}`) immediately return a rendering error instead of silently passing unrendered placeholders to production HTML.
- **Pre-Compiled Template AST (`gossr.MustCompile`)**: Pre-validates security context and compiles template ASTs at initialization time for ultra-fast binding (`.Bind(scope)`) and direct streaming (`.Render(writer, scope)`).
- **Component Recursion Protection**: Guards against infinite component loops with stack depth tracking across custom tags (`MaxRenderDepth = 100`).
- **Native HTTP Handlers & Error Propagation**: Serve components directly with `gossr.RenderHTTP(w, comp)` or `gossr.Handler(factoryFn)`, automatically setting `Content-Type: text/html; charset=utf-8` headers and calling a configurable `ErrorHandler` callback (HTTP 500) on render failures.
- **AHA Stack Native (ASTACK)**: Unescaped integration with **HTMX** out-of-band updates (`hx-delete`, `hx-target`, `hx-swap`) and **Alpine.js** client state (`x-data`, `x-show`, `@click`).

---

## ReactJS DX in Pure Go (Zero Node, Zero Transpilers)

GoSSR brings the familiar mental model of **ReactJS component-driven development** directly to backend Go code:

| ReactJS Concept | GoSSR Paradigm | How It Works |
| :--- | :--- | :--- |
| **Functional Components** | Go Functions returning `gossr.SSR` | Write pure Go functions that bind typed property structs to backtick HTML string templates. |
| **Component Props** | Go Structs (`UserProperties`) | Pass strongly-typed Go structs as properties into components. |
| **JSX Custom Component Tags** | `gossr.Register("UserCard", UserCard)` | Register component tags and use `<UserCard name="Sarah" />` directly inside HTML templates. |
| **`props.children` Composition**| `${properties.Children}` | Pass nested HTML or child `gossr.SSR` components to paired custom tags `<Card><h1>Content</h1></Card>`. |
| **Array Mapping (`items.map(...)`)**| `${properties.Items.map(item => ...)}` | Map over slices inside templates using `.map(item => <Item ... />)` lambdas. |
| **Ternary Conditionals** | `${properties.IsActive ? "active" : ""}` | Conditionally render inline attributes and classes with ternary expressions. |
| **Prop Typo Validation** | Strict Tag Attribute Validator | Automatically checks tag attributes against struct field names, rejecting typos (`nme="Sarah"`). |
| **Build Tools / Transpilers** | **Zero CLI Build Steps** | Run directly with native `go run` or `go build` with zero Node.js, Webpack, Babel, or `node_modules`. |

---

## Architecture & System Walkthrough

The repository is structured as a root library package with an `examples/` subpackage:

### Root Framework Package (`github.com/lemadane/gossr`)
- **[engine.go](file:///home/lem/Projects/go/GoSSR/engine.go)**: The generic reflection rendering engine implementing `gossr.SSR`, `gossr.RawHtml`, `gossr.SafeUrl`, `gossr.Register`, `gossr.RenderHTTP`, `gossr.Handler`, context-aware HTML scanner, custom tag parser, and `gossr.Render(templateString, scopeArguments...)`.
- **[http_helpers_test.go](file:///home/lem/Projects/go/GoSSR/http_helpers_test.go)**: Unit tests for `RenderHTTP`, `Handler`, exported HTML entity helpers (`EscapeHTML`, `UnescapeHTML`), and attribute typo validation.
- **[custom_tags_test.go](file:///home/lem/Projects/go/GoSSR/custom_tags_test.go)**: Unit tests for declarative custom component tags (self-closing, paired tags with `children`, struct & map factories).
- **[security_test.go](file:///home/lem/Projects/go/GoSSR/security_test.go)**: Unit tests for HTML context scanning, `RawHtml`, `SafeUrl` protocol sanitization, and recursion depth protection.
- **[forms_test.go](file:///home/lem/Projects/go/GoSSR/forms_test.go)**: Unit tests for form controls, attribute quote protection, checkboxes, and radio buttons.
- **[engine_test.go](file:///home/lem/Projects/go/GoSSR/engine_test.go)**: Core engine unit tests covering property substitution, map lambda evaluation, dollar sign preservation, and deep nested path resolution.

### Example Task Application (`examples/taskmanager/`)
- **[examples/taskmanager/store.go](file:///home/lem/Projects/go/GoSSR/examples/taskmanager/store.go)**: Thread-safe in-memory task store with mutex synchronization for full CRUD operations.
- **[examples/taskmanager/stats_component.go](file:///home/lem/Projects/go/GoSSR/examples/taskmanager/stats_component.go)**: Live dashboard statistics summary component (`Total`, `Pending`, `Completed`, `High Priority`).
- **[examples/taskmanager/form_component.go](file:///home/lem/Projects/go/GoSSR/examples/taskmanager/form_component.go)**: Collapsible creation form component with Alpine.js collapse (`x-data="{ expanded: false }"`), radio button priority selection, validation banners, and HTMX `hx-post="/api/tasks"` submission.
- **[examples/taskmanager/task_item.go](file:///home/lem/Projects/go/GoSSR/examples/taskmanager/task_item.go)**: Interactive task row component with HTMX `hx-put` toggle, `hx-delete` outerHTML swap, Alpine.js confirmation (`x-data="{ showConfirm: false }"`), and priority badges.
- **[examples/taskmanager/task_list.go](file:///home/lem/Projects/go/GoSSR/examples/taskmanager/task_list.go)**: Dashboard page component composing stats, live search input (`hx-get="/api/tasks/search" hx-trigger="keyup changed delay:250ms"`), filter tabs, creation form, and tasks list.
- **[examples/taskmanager/card.go](file:///home/lem/Projects/go/GoSSR/examples/taskmanager/card.go)**: Registered custom component tag wrapper `<TaskCard title="...">... </TaskCard>`.
- **[examples/taskmanager/main.go](file:///home/lem/Projects/go/GoSSR/examples/taskmanager/main.go)**: HTTP server serving `/tasks` page via `gossr.RenderHTTP` and REST HTMX endpoints (`/api/tasks`, `/api/tasks/search`, `/api/tasks/{id}`, `/api/tasks/{id}/toggle`).

---

## Security Model & Production Readiness Audit

GoSSR incorporates a multi-layer security architecture and contextual HTML parser to protect against XSS vulnerabilities, attribute breakouts, and executable script injections:

| Security Domain | Protection Mechanism | Status |
| :--- | :--- | :--- |
| **Normal Text Escaping** | All plain string interpolations pass through `html.EscapeString` | **Fixed** |
| **Attribute Quote Protection** | Double quote values in attributes are entity-escaped (`&quot;` / `&#34;`) | **Fixed** |
| **URL Protocol Auto-Sanitization** | Plain strings in URL attributes (`href`, `src`, `action`, etc.) with dangerous schemes (`javascript:`, `data:`) sanitize to `about:blank` | **Fixed** |
| **Explicit Trusted Wrappers** | `gossr.Raw(...)` and `gossr.URL(...)` explicitly mark trusted HTML content and sanitized URLs | **Implemented** |
| **Trusted SSR Components** | `gossr.SSR` component return values are preserved as trusted HTML structure | **Implemented** |
| **Dollar Sign (`$`) Preservation** | Pure string builder byte-slicing prevents regex `$1`, `$2` capture group corruption | **Fixed** |
| **Real `.map()` Lambda Execution** | Lambda body expressions (`item => <tag>${item.Val}</tag>`) are evaluated per item | **Fixed** |
| **Arbitrary Property Depth** | Recursive path traversal (`resolvePropertyPath`) supports N-level nested structs/pointers/maps | **Fixed** |
| **Strict Mode Validation** | `gossr.Strict(true)` raises rendering errors on unresolved property paths/typos | **Implemented** |
| **Direct Executable Contexts** | Scanner rejects `${...}` inside `<script>`, `<style>`, `<!-- -->`, `on*`, `style`, Alpine (`x-data`, `@click`), HTMX (`hx-on:*`) | **Protected** |
| **Map Lambda Security Contexts** | Character-by-character scanner enforces security context rules inside `.map(...)` lambdas | **Protected** |
| **Pre-Compiled Template AST** | `gossr.MustCompile` pre-validates template security context at `init()` time for ultra-fast binding | **Implemented** |
| **Stack Recursion Limit** | Custom component nesting depth tracked across tags (`MaxRenderDepth = 100`) | **Protected** |
| **Production CI Pipeline** | Multi-version matrix (`1.22`-`1.24`), `staticcheck`, `govulncheck`, `-race`, fuzzing, benchmarks | **Configured** |
| **Engine Performance** | Pre-compiled package regexes achieve **6.2 µs/render** and **8.3x speedup** | **Optimized** |

---

## Quickstart Guide

### 1. Register Custom JSX-Style Component Tags (`gossr.Register`)

```go
package main

import "github.com/lemadane/gossr"

type UserBadgeProps struct {
	Name string
	Role string
}

func UserBadge(props UserBadgeProps) gossr.SSR {
	return gossr.Render(`
		<span class="badge ${properties.Role}">${properties.Name} (${properties.Role})</span>
	`, props)
}

func init() {
	// Register custom component tag
	gossr.Register("UserBadge", UserBadge)
}

func UserList() gossr.SSR {
	return gossr.Render(`
		<div class="user-list">
			<UserBadge name="Sarah Connor" role="Admin" />
			<UserBadge name="Alex Mercer" role="Developer" />
		</div>
	`)
}
```

---

### 2. Native HTTP Handlers (`gossr.Handler` & `gossr.RenderHTTP`)

```go
package main

import (
	"fmt"
	"net/http"
	"github.com/lemadane/gossr"
)

func main() {
	// Option A: Use gossr.Handler wrapper
	http.HandleFunc("/user", gossr.Handler(func(r *http.Request) gossr.SSR {
		return UserForm(UserFormProperties{
			Name:      "Sarah Connor",
			Email:     "sarah@example.com",
			Subscribe: true,
			Role:      "ADMIN",
			Website:   gossr.URL("https://example.com"),
			BioHtml:   gossr.Raw("<b>Trusted Admin Bio</b>"),
		})
	}))

	// Option B: Use gossr.RenderHTTP directly inside standard handlers
	http.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		comp := TaskList(TaskListProperties{Title: "Deliverables"})
		_ = gossr.RenderHTTP(w, comp)
	})

	fmt.Println("Server running on http://localhost:8080/user")
	_ = http.ListenAndServe(":8080", nil)
}
```

---

### 3. Forms, Checkboxes & Radio Buttons

GoSSR handles form input value quote escaping, boolean checkbox state, and radio button option selection seamlessly using ternary expressions:

#### Component Definition (`user_form.go`)
```go
package main

import "github.com/lemadane/gossr"

type UserFormProperties struct {
	Name         string
	Email        string
	Subscribe    bool
	Role         string
	ErrorMessage string
}

func UserForm(props UserFormProperties) gossr.SSR {
	return gossr.Render(`
		<div class="form-container">
			${properties.ErrorMessage != "" ? "<p class=\"error\">" : ""}${properties.ErrorMessage}${properties.ErrorMessage != "" ? "</p>" : ""}
			
			<form hx-post="/user/profile" hx-target="#form-container">
				<!-- Text input with automatic quote escaping protection -->
				<label>Name:</label>
				<input type="text" name="name" value="${properties.Name}" />

				<label>Email:</label>
				<input type="email" name="email" value="${properties.Email}" />

				<!-- Checkbox boolean state rendering -->
				<label>
					<input type="checkbox" name="subscribe" ${properties.Subscribe ? "checked" : ""} />
					Subscribe to Newsletter
				</label>

				<!-- Radio buttons equality comparison rendering -->
				<label>Role:</label>
				<input type="radio" name="role" value="ADMIN" ${properties.Role == "ADMIN" ? "checked" : ""} /> Admin
				<input type="radio" name="role" value="USER" ${properties.Role == "USER" ? "checked" : ""} /> User

				<button type="submit">Save Profile</button>
			</form>
		</div>
	`, props)
}
```

#### Rendered HTML Output

```html
<div class="form-container">
	<p class="error">Email format is invalid &amp; missing domain</p>
	
	<form hx-post="/user/profile" hx-target="#form-container">
		<!-- Pre-filled text input with escaped double quotes -->
		<label>Name:</label>
		<input type="text" name="name" value="Sarah &quot;The Boss&quot; Connor" />

		<label>Email:</label>
		<input type="email" name="email" value="sarah@example.com" />

		<!-- Checked checkbox when Subscribe is true -->
		<label>
			<input type="checkbox" name="subscribe" checked />
			Subscribe to Newsletter
		</label>

		<!-- Checked radio button when Role == "ADMIN" -->
		<label>Role:</label>
		<input type="radio" name="role" value="ADMIN" checked /> Admin
		<input type="radio" name="role" value="USER"  /> User

		<button type="submit">Save Profile</button>
	</form>
</div>
```

---

### 4. HTMX & Alpine.js Integration (AHA Stack)

GoSSR preserves unescaped **HTMX** attributes (`hx-delete`, `hx-target`, `hx-swap`, `hx-post`) and **Alpine.js** directives (`x-data`, `x-show`, `@click`) for rich interactivity without build tools:

#### Component Definition (`task_item.go`)
```go
package main

import "github.com/lemadane/gossr"

type TaskItemProperties struct {
	Task Task
}

func TaskItem(properties TaskItemProperties) gossr.SSR {
	return gossr.Render(`
		<li id="task-${properties.Task.ID}" 
			class="task-item ${properties.Task.Completed ? "completed" : "pending"} priority-${properties.Task.Priority}"
			x-data="{ showConfirm: false }">
			
			<div class="task-info">
				<!-- HTMX toggle completion status -->
				<input type="checkbox" 
					   class="task-checkbox"
					   ${properties.Task.Completed ? "checked" : ""} 
					   hx-put="/api/tasks/${properties.Task.ID}/toggle" 
					   hx-target="#task-${properties.Task.ID}" 
					   hx-swap="outerHTML" />

				<span class="priority-badge ${properties.Task.Priority}">${properties.Task.Priority}</span>
				<span class="task-title ${properties.Task.Completed ? "line-through" : ""}">${properties.Task.Title}</span>
			</div>

			<div class="task-actions">
				<!-- Alpine.js toggle confirmation display -->
				<button class="button-toggle-delete" @click="showConfirm = !showConfirm">
					<span x-show="!showConfirm">Delete</span>
					<span x-show="showConfirm">Cancel</span>
				</button>

				<!-- HTMX out-of-band deletion request -->
				<button class="button-confirm-delete" 
						x-show="showConfirm" 
						@click.outside="showConfirm = false"
						hx-delete="/api/tasks/${properties.Task.ID}" 
						hx-target="#task-${properties.Task.ID}" 
						hx-swap="outerHTML">
					Confirm Delete?
				</button>
			</div>
		</li>
	`, properties)
}
```

#### Rendered HTML Output

```html
<li id="task-102" 
	class="task-item pending priority-HIGH"
	x-data="{ showConfirm: false }">
	
	<div class="task-info">
		<input type="checkbox" 
			   class="task-checkbox"
			   hx-put="/api/tasks/102/toggle" 
			   hx-target="#task-102" 
			   hx-swap="outerHTML" />

		<span class="priority-badge HIGH">HIGH</span>
		<span class="task-title ">Integrate HTMX out-of-band updates &amp; live search</span>
	</div>

	<div class="task-actions">
		<button class="button-toggle-delete" @click="showConfirm = !showConfirm">
			<span x-show="!showConfirm">Delete</span>
			<span x-show="showConfirm">Cancel</span>
		</button>

		<button class="button-confirm-delete" 
				x-show="showConfirm" 
				@click.outside="showConfirm = false"
				hx-delete="/api/tasks/102" 
				hx-target="#task-102" 
				hx-swap="outerHTML">
			Confirm Delete?
		</button>
	</div>
</li>
```

---

## Security & Architecture Comparison: GoSSR vs JSSR

| Feature | GoSSR | JSSR |
| :--- | :--- | :--- |
| **Component Model** | Pure Go functions returning `gossr.SSR` | Pure Java Records implementing `JssrComponent` |
| **Declarative Custom Tags** | `gossr.Register("Tag", Factory)` | `JssrComponent.register("Tag", Class)` |
| **Tag Types** | Self-closing (`<Tag />`) & paired (`<Tag>children</Tag>`) | Self-closing (`<Tag />`) & paired (`<Tag>children</Tag>`) |
| **Tag Attribute Validation** | Rejects unrecognized attributes / typos | Rejects unrecognized attributes / typos |
| **Default HTML Escaping** | Automatic (`html.EscapeString`) | Automatic (`escapeHtml`) |
| **Trusted HTML Wrapper** | `gossr.RawHtml` / `gossr.Raw("...")` | `RawHtml.of("...")` |
| **Safe URL Sanitizer** | `gossr.SafeUrl` / `gossr.URL("...")` | `SafeUrl.of("...")` |
| **Context Security Scanner** | Single-pass scanner enforcing context safety | Single-pass scanner enforcing context safety |
| **Forbidden Context Rejection**| Rejects `<script>`, `<style>`, `<!-- -->`, `on*`, `style` | Rejects `<script>`, `<style>`, `<!-- -->`, `on*`, `style` |
| **Attribute Quote Protection** | Quote escaping (`&#34;` / `&quot;`) | Quote escaping (`&quot;`) |
| **Recursion Depth Limit** | `MaxRenderDepth = 100` | `MAX_RENDER_DEPTH = 100` |
| **HTTP Response Helpers** | `gossr.RenderHTTP` & `gossr.Handler` | `JssrConverter` (Spring MVC Converter) |

---

## API Reference

### `gossr.SSR` Interface
```go
type SSR interface {
	Render(writer io.Writer) error
	String() string
}
```

### `gossr.MustCompile` & `gossr.Compile`
```go
type CompiledTemplate struct { ... }

func Compile(templateString string) (CompiledTemplate, error)
func MustCompile(templateString string) CompiledTemplate
func (ct CompiledTemplate) Bind(scopeArguments ...any) SSR
func (ct CompiledTemplate) Render(writer io.Writer, scopeArguments ...any) error
```

### `gossr.RenderHTTP` & `gossr.Handler`
```go
func RenderHTTP(w http.ResponseWriter, component SSR) error
func Handler(factory func(r *http.Request) SSR) http.HandlerFunc
```

### `gossr.Register` / `gossr.RegisterTag`
```go
func Register(tagName string, factory any)
```

### `gossr.EscapeHTML` & `gossr.UnescapeHTML`
```go
func EscapeHTML(input string) string
func UnescapeHTML(input string) string
```

---

## Testing & Verification Results

Run the full test suite across root library and example application:

```bash
go test -count=1 -v ./...
```

### Verified Test Output

```text
=== RUN   TestSelfClosingCustomTag
--- PASS: TestSelfClosingCustomTag (0.00s)
=== RUN   TestPairedCustomTagWithChildren
--- PASS: TestPairedCustomTagWithChildren (0.00s)
=== RUN   TestMapFactoryCustomTag
--- PASS: TestMapFactoryCustomTag (0.00s)
=== RUN   TestDynamicCustomTagAttributeEscaping
--- PASS: TestDynamicCustomTagAttributeEscaping (0.00s)
=== RUN   TestRawHtmlOfAndSafeUrlOfAndRegisterTag
--- PASS: TestRawHtmlOfAndSafeUrlOfAndRegisterTag (0.00s)
=== RUN   TestDirectWriterRenderingRawHtmlAndSafeUrl
--- PASS: TestDirectWriterRenderingRawHtmlAndSafeUrl (0.00s)
=== RUN   TestWriterFailuresInComponentRender
--- PASS: TestWriterFailuresInComponentRender (0.00s)
=== RUN   TestTruthinessEvaluationAllTypes
--- PASS: TestTruthinessEvaluationAllTypes (0.00s)
=== RUN   TestTernaryInequalityAndUnquoted
--- PASS: TestTernaryInequalityAndUnquoted (0.00s)
=== RUN   TestMapAndPointerPropertyResolutionEdgeCases
--- PASS: TestMapAndPointerPropertyResolutionEdgeCases (0.00s)
=== RUN   TestUnicodeAndSpecialCharsInPropertyNamesAndValues
--- PASS: TestUnicodeAndSpecialCharsInPropertyNamesAndValues (0.00s)
=== RUN   TestVeryLargeSliceMapping
--- PASS: TestVeryLargeSliceMapping (5.01s)
=== RUN   TestConcurrentRenderingThreadSafety
--- PASS: TestConcurrentRenderingThreadSafety (0.04s)
=== RUN   TestFormatAndEscapeValueDirectCall
--- PASS: TestFormatAndEscapeValueDirectCall (0.00s)
=== RUN   TestSimplePropertySubstitution
--- PASS: TestSimplePropertySubstitution (0.00s)
=== RUN   TestNestedPropertySubstitution
--- PASS: TestNestedPropertySubstitution (0.00s)
=== RUN   TestChildComponentRendering
--- PASS: TestChildComponentRendering (0.00s)
=== RUN   TestTernaryExpression
--- PASS: TestTernaryExpression (0.00s)
=== RUN   TestGenericSliceMapping
--- PASS: TestGenericSliceMapping (0.00s)
=== RUN   TestHTMLEscapingXSS
--- PASS: TestHTMLEscapingXSS (0.00s)
=== RUN   TestMapDollarSignPreservation
--- PASS: TestMapDollarSignPreservation (0.00s)
=== RUN   TestAllDollarSignVariationsInMap
--- PASS: TestAllDollarSignVariationsInMap (0.00s)
=== RUN   TestRealMapLambdaExecution
--- PASS: TestRealMapLambdaExecution (0.00s)
=== RUN   TestMapLambdaWithCustomComponentTag
--- PASS: TestMapLambdaWithCustomComponentTag (0.00s)
=== RUN   TestArbitraryNestedPropertyDepth
--- PASS: TestArbitraryNestedPropertyDepth (0.00s)
=== RUN   TestHandlerErrorPropagation
--- PASS: TestHandlerErrorPropagation (0.00s)
=== RUN   TestStrictModeUnresolvedPropertyError
--- PASS: TestStrictModeUnresolvedPropertyError (0.00s)
=== RUN   TestCompiledTemplateAndMustCompile
--- PASS: TestCompiledTemplateAndMustCompile (0.00s)
=== RUN   TestRawHtmlInsideAttributeIsEscaped
--- PASS: TestRawHtmlInsideAttributeIsEscaped (0.00s)
=== RUN   TestNestedChildComponentErrorPropagation
--- PASS: TestNestedChildComponentErrorPropagation (0.00s)
=== RUN   TestNonStringKeyedMapNoPanic
--- PASS: TestNonStringKeyedMapNoPanic (0.00s)
=== RUN   TestFormInputAttributeQuoteProtection
--- PASS: TestFormInputAttributeQuoteProtection (0.00s)
=== RUN   TestCheckboxAndRadioButtonRendering
--- PASS: TestCheckboxAndRadioButtonRendering (0.00s)
=== RUN   TestFormSubmissionValidationErrorFeedback
--- PASS: TestFormSubmissionValidationErrorFeedback (0.00s)
=== RUN   TestExportedHTMLEntityHelpers
--- PASS: TestExportedHTMLEntityHelpers (0.00s)
=== RUN   TestRenderHTTP
--- PASS: TestRenderHTTP (0.00s)
=== RUN   TestHTTPHandlerWrapper
--- PASS: TestHTTPHandlerWrapper (0.00s)
=== RUN   TestCustomTagAttributeTypoRejection
--- PASS: TestCustomTagAttributeTypoRejection (0.00s)
=== RUN   TestSecurityContextRejectionScript
--- PASS: TestSecurityContextRejectionScript (0.00s)
=== RUN   TestSecurityContextRejectionStyle
--- PASS: TestSecurityContextRejectionStyle (0.00s)
=== RUN   TestSecurityContextRejectionComment
--- PASS: TestSecurityContextRejectionComment (0.00s)
=== RUN   TestSecurityContextRejectionInlineEventHandler
--- PASS: TestSecurityContextRejectionInlineEventHandler (0.00s)
=== RUN   TestSecurityContextRejectionInlineStyleAttr
--- PASS: TestSecurityContextRejectionInlineStyleAttr (0.00s)
=== RUN   TestRawHtmlWrapper
--- PASS: TestRawHtmlWrapper (0.00s)
=== RUN   TestSafeUrlSanitizer
--- PASS: TestSafeUrlSanitizer (0.00s)
=== RUN   TestNilPropertyHandling
--- PASS: TestNilPropertyHandling (0.00s)
=== RUN   TestNormalStringInUrlAttributeAutoSanitized
--- PASS: TestNormalStringInUrlAttributeAutoSanitized (0.00s)
=== RUN   TestExecutableAttributeRejectionAlpineAndHTMX
--- PASS: TestExecutableAttributeRejectionAlpineAndHTMX (0.00s)
=== RUN   TestComponentRecursionDepthProtection
--- PASS: TestComponentRecursionDepthProtection (0.00s)
=== RUN   TestCustomTagPropsReflectionNoPanic
--- PASS: TestCustomTagPropsReflectionNoPanic (0.00s)
=== RUN   TestMapLambdaSecurityContextRejectionAndSanitization
--- PASS: TestMapLambdaSecurityContextRejectionAndSanitization (0.00s)
=== RUN   FuzzRender
--- PASS: FuzzRender (0.00s)
=== RUN   FuzzSanitizeUrl
--- PASS: FuzzSanitizeUrl (0.00s)
PASS
ok      github.com/lemadane/gossr       5.058s
=== RUN   TestE2ETaskPageEndpoint
--- PASS: TestE2ETaskPageEndpoint (0.01s)
=== RUN   TestE2ECreateTaskEndpoint
--- PASS: TestE2ECreateTaskEndpoint (0.02s)
=== RUN   TestE2EToggleTaskEndpoint
--- PASS: TestE2EToggleTaskEndpoint (0.00s)
=== RUN   TestE2EDeleteTaskEndpoint
--- PASS: TestE2EDeleteTaskEndpoint (0.00s)
=== RUN   TestE2ESearchEndpoint
--- PASS: TestE2ESearchEndpoint (0.00s)
=== RUN   TestE2ESecurityAndDollarPreservation
--- PASS: TestE2ESecurityAndDollarPreservation (0.00s)
=== RUN   TestE2EHTTPHandlerAndCustomTags
--- PASS: TestE2EHTTPHandlerAndCustomTags (0.00s)
=== RUN   TestE2EFormControlsCheckboxesRadio
--- PASS: TestE2EFormControlsCheckboxesRadio (0.00s)
=== RUN   TestE2EUrlAutoSanitizationInHttp
--- PASS: TestE2EUrlAutoSanitizationInHttp (0.00s)
=== RUN   TestE2EExecutableAttributeRejectionInHttp
--- PASS: TestE2EExecutableAttributeRejectionInHttp (0.00s)
=== RUN   TestE2ERecursionProtectionInHttp
--- PASS: TestE2ERecursionProtectionInHttp (0.00s)
=== RUN   TestE2EDynamicCustomTagEscapingInHttp
--- PASS: TestE2EDynamicCustomTagEscapingInHttp (0.00s)
=== RUN   TestE2ECustomTagReflectionPropsInHttp
--- PASS: TestE2ECustomTagReflectionPropsInHttp (0.00s)
PASS
ok      github.com/lemadane/gossr/examples/taskmanager  0.049s
```

---

## Continuous Integration & Security Auditing

GoSSR includes a production GitHub Actions CI pipeline ([.github/workflows/ci.yml](file:///home/lem/Projects/go/GoSSR/.github/workflows/ci.yml)) configured for every push and pull request:
- **Go Version Matrix**: Automated testing across Go `1.22.x`, `1.23.x`, and `1.24.x`.
- **Race Detection**: `go test -v -race ./...` for data race safety.
- **Code Formatting & Linting**: Strict `gofmt -l .` and `go vet ./...` checks.
- **Static Analysis**: `staticcheck ./...` for code quality and correctness.
- **Vulnerability Scanning**: `govulncheck ./...` for security advisory checking.
- **Fuzzing**: Native `go test -fuzz=...` verification.
- **Benchmark Regression Protection**: `go test -bench=. -benchmem ./...`.

---

## Open Source Governance & Community

- **[SECURITY.md](file:///home/lem/Projects/go/GoSSR/SECURITY.md)**: Vulnerability reporting guidelines and private advisory disclosure process.
- **[CONTRIBUTING.md](file:///home/lem/Projects/go/GoSSR/CONTRIBUTING.md)**: Guidelines for bug reports, PR standards, allocation constraints, and code style.
- **[CHANGELOG.md](file:///home/lem/Projects/go/GoSSR/CHANGELOG.md)**: Complete version history and release notes starting from `v0.1.0`.

---

## License
MIT License
