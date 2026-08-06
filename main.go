package main

import (
	"fmt"
	"net/http"
)

func handleTaskPage(responseWriter http.ResponseWriter, request *http.Request) {
	sampleTasks := []Task{
		{ID: "101", Title: "Initialize pure GoSSR engine", Completed: true},
		{ID: "102", Title: "Integrate HTMX out-of-band updates", Completed: false},
		{ID: "103", Title: "Attach AlpineJS client interactivity", Completed: false},
	}

	taskListComponent := TaskList(TaskListProperties{
		Title:    "Project Deliverables",
		TaskList: sampleTasks,
	})

	cardWrapperComponent := Card(CardProperties{
		Title:    "Active Tasks Summary",
		Children: taskListComponent,
	})

	fullHtmlDocument := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>GoSSR Task Manager</title>
	<script src="https://unpkg.com/htmx.org@1.9.10"></script>
	<script defer src="https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js"></script>
	<style>
		:root {
			--background-color: #0f172a;
			--card-background: #1e293b;
			--text-primary: #f8fafc;
			--text-secondary: #94a3b8;
			--accent-color: #38bdf8;
			--danger-color: #ef4444;
			--success-color: #10b981;
			--border-color: #334155;
		}

		body {
			font-family: 'Inter', system-ui, -apple-system, sans-serif;
			background-color: var(--background-color);
			color: var(--text-primary);
			margin: 0;
			padding: 2rem;
			display: flex;
			justify-content: center;
		}

		.page-container {
			width: 100%%;
			max-width: 720px;
		}

		.page-header h1 {
			font-size: 2rem;
			margin-bottom: 1.5rem;
			color: var(--accent-color);
		}

		.card-container {
			background-color: var(--card-background);
			border: 1px solid var(--border-color);
			border-radius: 12px;
			padding: 1.5rem;
			box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.3);
		}

		.card-header {
			display: flex;
			justify-content: space-between;
			align-items: center;
			margin-bottom: 1rem;
		}

		.card-header h3 {
			margin: 0;
			font-size: 1.25rem;
		}

		.button-toggle, .button-delete {
			background: transparent;
			border: 1px solid var(--border-color);
			color: var(--text-primary);
			padding: 0.5rem 1rem;
			border-radius: 6px;
			cursor: pointer;
			font-size: 0.875rem;
			transition: all 0.2s ease;
		}

		.button-toggle:hover {
			border-color: var(--accent-color);
			color: var(--accent-color);
		}

		.button-delete {
			border-color: var(--danger-color);
			color: var(--danger-color);
		}

		.button-delete:hover {
			background-color: var(--danger-color);
			color: white;
		}

		.task-list {
			list-style: none;
			padding: 0;
			margin: 0;
		}

		.task-item {
			display: flex;
			justify-content: space-between;
			align-items: center;
			padding: 0.875rem 1rem;
			border-bottom: 1px solid var(--border-color);
			transition: opacity 0.3s ease;
		}

		.task-item:last-child {
			border-bottom: none;
		}

		.task-info {
			display: flex;
			align-items: center;
			gap: 0.75rem;
		}

		.task-badge {
			font-size: 0.75rem;
			padding: 0.25rem 0.5rem;
			border-radius: 9999px;
			font-weight: 600;
		}

		.task-item.completed .task-badge {
			background-color: rgba(16, 185, 129, 0.2);
			color: var(--success-color);
		}

		.task-item.pending .task-badge {
			background-color: rgba(56, 189, 248, 0.2);
			color: var(--accent-color);
		}

		.task-title {
			font-size: 1rem;
		}
	</style>
</head>
<body>
	%s
</body>
</html>`, cardWrapperComponent.String())

	responseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = responseWriter.Write([]byte(fullHtmlDocument))
}

func handleDeleteTask(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodDelete {
		responseWriter.WriteHeader(http.StatusOK)
		return
	}
	http.Error(responseWriter, "Method not allowed", http.StatusMethodNotAllowed)
}

func setupRoutes() *http.ServeMux {
	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/tasks", handleTaskPage)
	serverMux.HandleFunc("/api/tasks/", handleDeleteTask)
	return serverMux
}

func main() {
	serverMux := setupRoutes()
	fmt.Println("Server running on http://localhost:8080/tasks")
	_ = http.ListenAndServe(":8080", serverMux)
}
