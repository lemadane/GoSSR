# GoSSR ⚡

**GoSSR** is a lightweight, pure, zero-transpilation Server-Side Rendering (SSR) framework for Go that brings a React-like Developer Experience (DX) directly to native Go backtick string templates.

With **GoSSR**, you write component-based user interfaces in pure `.go` files with **zero external build steps, zero node_modules, zero transpilers, and zero code generators**.

---

## ✨ Key Features

- **React-like DX in Pure Go**: Define reusable UI components using standard Go functions returning the `SSR` interface.
- **Zero CLI Build Steps**: Runs directly with standard `go run` or `go build` with zero node_modules or transpilers.
- **Template Expression Parsing**:
  - `${properties.FieldName}`: Top-level property evaluation.
  - `${properties.Parent.Child}`: Nested struct property resolution.
  - `${properties.Children}`: Embedded child `SSR` component rendering.
  - `${properties.Condition ? "OptionA" : "OptionB"}`: Inline ternary conditional evaluation.
  - `${properties.Slice.map(item => Component(...))}`: Slice mapping for list rendering.
- **AHA Stack Native (ASTACK)**: Unescaped integration with **HTMX** out-of-band updates (`hx-delete`, `hx-target`, `hx-swap`) and **Alpine.js** client state (`x-data`, `x-show`, `@click`).
- **High-Performance Stream Rendering**: Render directly to `io.Writer` or `http.ResponseWriter` with `Render(writer)`.

---

## 🏗️ Architecture & System Walkthrough

The GoSSR architecture is built around decoupled, single-responsibility Go modules:

- **[engine.go](file:///home/lem/Projects/go/GoSSR/engine.go)**: The reflection rendering engine implementing the `SSR` interface and `Render(templateString, scopeArguments...)` constructor helper. Safely evaluates top-level properties, nested struct fields (`Task.ID`), ternary expressions (`Completed ? "Done" : "Pending"`), child `SSR` components, and slice mapping expressions while ignoring unexported struct fields.
- **[card.go](file:///home/lem/Projects/go/GoSSR/card.go)**: Reusable container wrapper accepting `Children SSR` and Title, styled with Alpine.js collapsible client state (`x-data="{ collapsed: false }"`).
- **[task_item.go](file:///home/lem/Projects/go/GoSSR/task_item.go)**: Leaf item component integrating HTMX out-of-band deletion (`hx-delete`, `hx-target`, `hx-swap`) and Alpine.js inline deletion confirmation.
- **[task_list.go](file:///home/lem/Projects/go/GoSSR/task_list.go)**: Page component composing task controls and rendering array mapping over `TaskList` slice items.
- **[main.go](file:///home/lem/Projects/go/GoSSR/main.go)**: HTTP server exposing `/tasks` page and `/api/tasks/{id}` endpoint with embedded dark mode CSS styling and HTMX / AlpineJS CDN dependencies.

---

## 🚀 Quickstart Guide

### 1. Define a Component

Every component in **GoSSR** is a standard Go function returning the `SSR` interface via the `Render(...)` helper:

```go
package main

type CardProperties struct {
	Title    string
	Children SSR
}

func Card(properties CardProperties) SSR {
	return Render(`
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

type Task struct {
	ID        string
	Title     string
	Completed bool
}

type TaskItemProperties struct {
	Task Task
}

func TaskItem(properties TaskItemProperties) SSR {
	return Render(`
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

### 3. List Mapping & Array Rendering

Render dynamic arrays using the `${properties.SliceName.map(...)}` syntax:

```go
package main

type TaskListProperties struct {
	Title    string
	TaskList []Task
}

func TaskList(properties TaskListProperties) SSR {
	return Render(`
		<main class="page-container">
			<header class="page-header">
				<h1>${properties.Title}</h1>
			</header>

			<div class="task-controls">
				<ul class="task-list">
					${properties.TaskList.map(singleTask => TaskItem(TaskItemProperties{Task: singleTask}))}
				</ul>
			</div>
		</main>
	`, properties)
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

## 🛠️ API Reference

### `SSR` Interface
```go
type SSR interface {
	Render(writer io.Writer) error
	String() string
}
```

### `Render()` Helper
```go
func Render(templateString string, scopeArguments ...any) SSR
```
Constructs a component by binding a backtick template string to scope property structs.

---

## 🧪 Testing & Verification Results

GoSSR includes a comprehensive unit test suite (`engine_test.go`) and end-to-end integration test suite (`e2e_test.go`).

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
=== RUN   TestSimplePropertySubstitution
--- PASS: TestSimplePropertySubstitution (0.00s)
=== RUN   TestNestedPropertySubstitution
--- PASS: TestNestedPropertySubstitution (0.00s)
=== RUN   TestChildComponentRendering
--- PASS: TestChildComponentRendering (0.00s)
=== RUN   TestTernaryExpression
--- PASS: TestTernaryExpression (0.00s)
=== RUN   TestSliceMapping
--- PASS: TestSliceMapping (0.00s)
PASS
ok      GoSSR   0.007s
```

All 7 test cases pass cleanly, verifying property evaluation, nested struct fields, child component composition, ternary logic, slice mappings, HTTP page rendering, and HTMX out-of-band endpoints.

---

## 📁 Repository Structure

```
.
├── engine.go       # Reflection rendering engine & SSR interface
├── card.go         # Card component wrapper
├── task_item.go    # TaskItem leaf component with HTMX & Alpine.js
├── task_list.go    # TaskList parent component & list mapper
├── main.go         # HTTP server and route handlers
├── engine_test.go  # Unit test suite
├── e2e_test.go     # End-to-end integration test suite
├── README.md       # Framework documentation & quickstart
└── .gitignore      # Git ignore rules
```

---

## 📄 License
MIT License
