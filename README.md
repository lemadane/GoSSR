# GoSSR

**GoSSR** is a lightweight, pure, zero-transpilation Server-Side Rendering (SSR) framework for Go that brings a React-like Developer Experience (DX) directly to native Go backtick string templates.

With **GoSSR**, you write component-based user interfaces in pure `.go` files with **zero external build steps, zero node_modules, zero transpilers, and zero code generators**.

---

## Key Features

- **Clean & Reusable Go Package**: Importable cleanly as `import "github.com/lemadane/gossr"`.
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

The repository is structured as a root library package with an `examples/` subpackage:

### Root Framework Package (`github.com/lemadane/gossr`)
- **[engine.go](file:///home/lem/Projects/go/GoSSR/engine.go)**: The 100% generic reflection rendering engine at repository root implementing `gossr.SSR` and `gossr.Render(templateString, scopeArguments...)`.

### Example Task Application (`examples/taskmanager/`)
- **[examples/taskmanager/card.go](file:///home/lem/Projects/go/GoSSR/examples/taskmanager/card.go)**: Container wrapper component accepting `Children gossr.SSR`.
- **[examples/taskmanager/task_item.go](file:///home/lem/Projects/go/GoSSR/examples/taskmanager/task_item.go)**: Leaf item component integrating HTMX out-of-band deletion and Alpine.js confirmation.
- **[examples/taskmanager/task_list.go](file:///home/lem/Projects/go/GoSSR/examples/taskmanager/task_list.go)**: Page component composing task controls and rendering `[]gossr.SSR` slice items.
- **[examples/taskmanager/main.go](file:///home/lem/Projects/go/GoSSR/examples/taskmanager/main.go)**: HTTP server exposing `/tasks` page and `/api/tasks/{id}` endpoint with embedded dark mode styling.

---

## Quickstart Guide

### 1. Import Package & Define a Component

```go
package main

import "github.com/lemadane/gossr"

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

```go
package main

import "github.com/lemadane/gossr"

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

```go
package main

import "github.com/lemadane/gossr"

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

Run the test suite across root library and example application:

```bash
go test -v ./...
```

### Verified Test Output

```text
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
ok      github.com/lemadane/gossr       0.003s
=== RUN   TestE2ETaskPageEndpoint
--- PASS: TestE2ETaskPageEndpoint (0.00s)
=== RUN   TestE2EDeleteTaskEndpoint
--- PASS: TestE2EDeleteTaskEndpoint (0.00s)
PASS
ok      github.com/lemadane/gossr/examples/taskmanager  0.005s
```

---

## Repository Structure

```
.
├── engine.go                 # Root framework rendering engine & gossr.SSR interface
├── engine_test.go            # Unit test suite for root framework
├── go.mod                    # Module definition (github.com/lemadane/gossr)
├── README.md                 # Framework documentation & quickstart
├── .gitignore                # Git ignore rules
└── examples/
    └── taskmanager/          # Example web application using gossr
        ├── main.go
        ├── card.go
        ├── task_item.go
        ├── task_list.go
        └── e2e_test.go
```

---

## License
MIT License
