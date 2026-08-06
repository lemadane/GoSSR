# GoSSR

**GoSSR** is a lightweight, pure, zero-transpilation Server-Side Rendering (SSR) framework for Go that brings a React-like Developer Experience (DX) directly to native Go backtick string templates.

With **GoSSR**, you write component-based user interfaces in pure `.go` files with **zero external build steps, zero node_modules, zero transpilers, and zero code generators**.

---

## Key Features

- **100% Generic & Reusable Go Package**: Importable as `import "GoSSR/gossr"` (or `import "github.com/yourusername/gossr"`).
- **React-like DX in Pure Go**: Define reusable UI components using standard Go functions returning the `gossr.SSR` interface.
- **Zero CLI Build Steps**: Runs directly with standard `go run` or `go build` with zero node_modules or transpilers.
- **Template Expression Parsing**:
  - `${properties.FieldName}`: Top-level property evaluation.
  - `${properties.Parent.Child}`: Nested struct property resolution.
  - `${properties.Children}`: Embedded child `gossr.SSR` component rendering.
  - `${properties.Condition ? "OptionA" : "OptionB"}`: Inline ternary conditional evaluation.
  - `${properties.Slice.map(item => Component(...))}`: Generic slice mapping for list rendering (`[]gossr.SSR`, `[]fmt.Stringer`, `[]any`).
- **AHA Stack Native (ASTACK)**: Unescaped integration with **HTMX** out-of-band updates (`hx-delete`, `hx-target`, `hx-swap`) and **Alpine.js** client state (`x-data`, `x-show`, `@click`).
- **High-Performance Stream Rendering**: Render directly to `io.Writer` or `http.ResponseWriter` with `gossr.Render(writer)`.

---

## Architecture & System Walkthrough

The GoSSR architecture separates the reusable core rendering engine from domain-specific application components:

### Reusable Engine Package (`gossr/`)
- **[gossr/engine.go](file:///home/lem/Projects/go/GoSSR/gossr/engine.go)**: The 100% generic reflection rendering engine implementing the `gossr.SSR` interface and `gossr.Render(templateString, scopeArguments...)` constructor. Evaluates properties, nested struct fields, child `SSR` components, ternary logic, and slice mappings with zero hardcoded domain dependencies.

### Application Components (`main/`)
- **[card.go](file:///home/lem/Projects/go/GoSSR/card.go)**: Container wrapper component accepting `Children gossr.SSR` and Title, styled with Alpine.js collapsible client state (`x-data="{ collapsed: false }"`).
- **[task_item.go](file:///home/lem/Projects/go/GoSSR/task_item.go)**: Leaf item component integrating HTMX out-of-band deletion (`hx-delete`, `hx-target`, `hx-swap`) and Alpine.js inline deletion confirmation.
- **[task_list.go](file:///home/lem/Projects/go/GoSSR/task_list.go)**: Page component composing task controls and rendering `[]gossr.SSR` slice items.
- **[main.go](file:///home/lem/Projects/go/GoSSR/main.go)**: HTTP server exposing `/tasks` page and `/api/tasks/{id}` endpoint with embedded dark mode CSS styling and HTMX / AlpineJS CDN dependencies.

---

## Quickstart Guide

### 1. Import Package & Define a Component

Every component in **GoSSR** is a standard Go function returning `gossr.SSR` via `gossr.Render(...)`:

```go
package main

import "GoSSR/gossr"

type CardProperties struct {
	Title    string
	Children gossr.SSR
}

func Card(properties CardProperties) gossr.SSR {
	return gossr.Render(`
		<div class="card-container" x-data="{ collapsed: false }">
			<header class="card-header">
				<h3>${properties.Title}</h3>
				<button @click="collapsed = !collapsed" class="button-toggle">
					<span x-show="!collapsed">Collapse</span>
					<span x-show="collapsed">Expand</span>
				</button>
			</header>

			<div x-show="!collapsed" class="card-body">
				${properties.Children}
			</div>
		</div>
	`, properties)
}
```

---

### 2. Leaf Components & AHA Stack Interactivity

Combine HTMX and Alpine.js inside backtick templates:

```go
package main

import "GoSSR/gossr"

type Task struct {
	ID        string
	Title     string
	Completed bool
}

type TaskItemProperties struct {
	Task Task
}

func TaskItem(properties TaskItemProperties) gossr.SSR {
	return gossr.Render(`
		<li id="task-${properties.Task.ID}" class="task-item ${properties.Task.Completed ? "completed" : "pending"}">
			<div class="task-info">
				<span class="task-badge">${properties.Task.Completed ? "Completed" : "Pending"}</span>
				<span class="task-title">${properties.Task.Title}</span>
			</div>

			<!-- HTMX deletes task on server; Alpine handles click confirmation state -->
			<button 
				hx-delete="/api/tasks/${properties.Task.ID}"
				hx-target="#task-${properties.Task.ID}"
				hx-swap="outerHTML"
				x-data="{ confirming: false }"
				@click="if (!confirming) { confirming = true; event.preventDefault(); }"
				class="button-delete"
			>
				<span x-show="!confirming">Delete</span>
				<span x-show="confirming">Confirm Delete?</span>
			</button>
		</li>
	`, properties)
}
```

---

### 3. Generic List Mapping & Array Rendering

Render dynamic arrays by mapping slice items into `[]gossr.SSR`:

```go
package main

import "GoSSR/gossr"

type TaskListProperties struct {
	Title    string
	TaskList []Task
}

func TaskList(properties TaskListProperties) gossr.SSR {
	renderedTaskItems := make([]gossr.SSR, len(properties.TaskList))
	for index, singleTask := range properties.TaskList {
		renderedTaskItems[index] = TaskItem(TaskItemProperties{Task: singleTask})
	}

	type TaskListRenderProperties struct {
		Title string
		Tasks []gossr.SSR
	}

	return gossr.Render(`
		<main class="page-container">
			<header class="page-header">
				<h1>${properties.Title}</h1>
			</header>

			<div class="task-controls">
				<ul class="task-list">
					${properties.Tasks.map(singleTask => TaskItem(TaskItemProperties{Task: singleTask}))}
				</ul>
			</div>
		</main>
	`, TaskListRenderProperties{
		Title: properties.Title,
		Tasks: renderedTaskItems,
	})
}
```

---

### 4. Serve HTTP Requests

Render components directly into standard Go `http.ResponseWriter`:

```go
package main

import (
	"fmt"
	"net/http"

	"GoSSR/gossr"
)

func handleTaskPage(responseWriter http.ResponseWriter, request *http.Request) {
	sampleTasks := []Task{
		{ID: "101", Title: "Initialize pure GoSSR engine", Completed: true},
		{ID: "102", Title: "Integrate HTMX out-of-band updates", Completed: false},
	}

	pageComponent := TaskList(TaskListProperties{
		Title:    "Project Deliverables",
		TaskList: sampleTasks,
	})

	responseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pageComponent.Render(responseWriter)
}

func main() {
	http.HandleFunc("/tasks", handleTaskPage)
	fmt.Println("Server running on http://localhost:8080/tasks")
	_ = http.ListenAndServe(":8080", nil)
}
```

---

## API Reference

### `gossr.SSR` Interface
```go
type SSR interface {
	Render(writer io.Writer) error
	String() string
}
```

### `gossr.Render()` Helper
```go
func Render(templateString string, scopeArguments ...any) SSR
```
Constructs a component by binding a backtick template string to scope property structs.

---

## Testing & Verification Results

GoSSR includes test suites for both the engine subpackage (`gossr/engine_test.go`) and application E2E routes (`e2e_test.go`).

Run the test suite with:

```bash
go test -v ./...
```

### Verified Test Output

```text
=== RUN   TestE2ETaskPageEndpoint
--- PASS: TestE2ETaskPageEndpoint (0.00s)
=== RUN   TestE2EDeleteTaskEndpoint
--- PASS: TestE2EDeleteTaskEndpoint (0.00s)
PASS
ok      GoSSR   0.006s
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
PASS
ok      GoSSR/gossr     0.004s
```

All test cases pass cleanly across both packages.

---

## Repository Structure

```
.
├── gossr/
│   ├── engine.go       # Reusable reflection rendering engine & gossr.SSR interface
│   └── engine_test.go  # Unit test suite for gossr package
├── card.go             # Card component wrapper
├── task_item.go        # TaskItem leaf component with HTMX & Alpine.js
├── task_list.go        # TaskList parent component & list mapper
├── main.go             # HTTP server and route handlers
├── e2e_test.go         # End-to-end integration test suite
├── README.md           # Framework documentation & quickstart
└── .gitignore          # Git ignore rules
```

---

## License
MIT License
