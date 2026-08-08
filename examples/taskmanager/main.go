package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/lemadane/gossr"
)

var globalTaskStore = NewTaskStore()

func handleTaskPage(responseWriter http.ResponseWriter, request *http.Request) {
	allTasks := globalTaskStore.GetAllTasks()
	stats := globalTaskStore.GetStats()

	taskListComponent := TaskList(TaskListProperties{
		Title:       "Task Manager",
		Tasks:       allTasks,
		Stats:       stats,
		SearchQuery: "",
		FilterTab:   "all",
	})

	cardWrapperComponent := Card(CardProperties{
		Title:    "Dashboard Summary",
		Children: taskListComponent,
	})

	fullHtmlDocument := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>GoSSR Task Manager (AHA Stack)</title>
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
			--warning-color: #f59e0b;
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
			max-width: 800px;
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

		.stats-grid {
			display: grid;
			grid-template-columns: repeat(4, 1fr);
			gap: 1rem;
			margin-bottom: 1.5rem;
		}

		.stat-card {
			background-color: rgba(255, 255, 255, 0.03);
			border: 1px solid var(--border-color);
			padding: 1rem;
			border-radius: 8px;
			text-align: center;
		}

		.stat-value {
			display: block;
			font-size: 1.5rem;
			font-weight: 700;
			color: var(--text-primary);
		}

		.stat-label {
			font-size: 0.75rem;
			color: var(--text-secondary);
			text-transform: uppercase;
			letter-spacing: 0.05em;
		}

		.stat-card.completed .stat-value { color: var(--success-color); }
		.stat-card.pending .stat-value { color: var(--accent-color); }
		.stat-card.high-priority .stat-value { color: var(--danger-color); }

		.task-form-wrapper {
			background: rgba(255, 255, 255, 0.02);
			border: 1px solid var(--border-color);
			border-radius: 8px;
			padding: 1rem;
			margin-bottom: 1.5rem;
		}

		.form-header {
			display: flex;
			justify-content: space-between;
			align-items: center;
		}

		.form-header h3 {
			margin: 0;
			font-size: 1.1rem;
		}

		.form-group {
			margin-top: 1rem;
		}

		.form-group label {
			display: block;
			margin-bottom: 0.35rem;
			font-size: 0.875rem;
			color: var(--text-secondary);
		}

		.input-text, .input-search {
			width: 100%%;
			box-sizing: border-box;
			padding: 0.6rem 0.8rem;
			background-color: var(--background-color);
			border: 1px solid var(--border-color);
			border-radius: 6px;
			color: var(--text-primary);
			font-size: 0.9rem;
		}

		.radio-group {
			display: flex;
			gap: 1rem;
		}

		.radio-label {
			font-size: 0.875rem;
			cursor: pointer;
		}

		.button-submit, .button-toggle, .button-toggle-delete, .button-confirm-delete {
			background: transparent;
			border: 1px solid var(--border-color);
			color: var(--text-primary);
			padding: 0.4rem 0.8rem;
			border-radius: 6px;
			cursor: pointer;
			font-size: 0.85rem;
			transition: all 0.2s ease;
		}

		.button-submit {
			margin-top: 1rem;
			background-color: var(--accent-color);
			color: var(--background-color);
			border: none;
			font-weight: 600;
		}

		.button-toggle-delete {
			border-color: var(--border-color);
		}

		.button-confirm-delete {
			border-color: var(--danger-color);
			background-color: var(--danger-color);
			color: white;
		}

		.search-filter-bar {
			display: flex;
			justify-content: space-between;
			align-items: center;
			gap: 1rem;
			margin-bottom: 1.5rem;
		}

		.search-input-wrapper {
			flex: 1;
		}

		.filter-tabs {
			display: flex;
			gap: 0.25rem;
			background: var(--background-color);
			padding: 0.25rem;
			border-radius: 6px;
			border: 1px solid var(--border-color);
		}

		.filter-tab {
			background: transparent;
			border: none;
			color: var(--text-secondary);
			padding: 0.35rem 0.75rem;
			border-radius: 4px;
			cursor: pointer;
			font-size: 0.8rem;
		}

		.filter-tab.active {
			background-color: var(--card-background);
			color: var(--accent-color);
			font-weight: 600;
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
		}

		.task-item:last-child {
			border-bottom: none;
		}

		.task-info {
			display: flex;
			align-items: center;
			gap: 0.75rem;
		}

		.priority-badge {
			font-size: 0.7rem;
			padding: 0.2rem 0.5rem;
			border-radius: 9999px;
			font-weight: 700;
		}

		.priority-badge.HIGH { background: rgba(239, 68, 68, 0.2); color: var(--danger-color); }
		.priority-badge.MEDIUM { background: rgba(245, 158, 11, 0.2); color: var(--warning-color); }
		.priority-badge.LOW { background: rgba(16, 185, 129, 0.2); color: var(--success-color); }

		.task-title.line-through {
			text-decoration: line-through;
			color: var(--text-secondary);
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

func handleCreateTask(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(responseWriter, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_ = request.ParseForm()
	title := request.FormValue("title")
	priority := request.FormValue("priority")

	if strings.TrimSpace(title) != "" {
		globalTaskStore.CreateTask(title, priority, false)
	}

	allTasks := globalTaskStore.GetAllTasks()
	stats := globalTaskStore.GetStats()

	taskListComponent := TaskList(TaskListProperties{
		Title:     "Task Manager",
		Tasks:     allTasks,
		Stats:     stats,
		FilterTab: "all",
	})

	_ = gossr.RenderHTTP(responseWriter, taskListComponent)
}

func handleSearchTasks(responseWriter http.ResponseWriter, request *http.Request) {
	query := request.URL.Query().Get("q")
	filteredTasks := globalTaskStore.SearchTasks(query, "all")
	stats := globalTaskStore.GetStats()

	taskListComponent := TaskList(TaskListProperties{
		Title:       "Task Manager",
		Tasks:       filteredTasks,
		Stats:       stats,
		SearchQuery: query,
		FilterTab:   "all",
	})

	_ = gossr.RenderHTTP(responseWriter, taskListComponent)
}

func handleTaskMutation(responseWriter http.ResponseWriter, request *http.Request) {
	path := request.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) < 3 {
		http.Error(responseWriter, "Invalid endpoint path", http.StatusBadRequest)
		return
	}

	taskID := parts[2]

	if request.Method == http.MethodDelete {
		if globalTaskStore.DeleteTask(taskID) {
			responseWriter.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(responseWriter, request)
		return
	}

	if request.Method == http.MethodPut && len(parts) >= 4 && parts[3] == "toggle" {
		updatedTask, found := globalTaskStore.ToggleTask(taskID)
		if found {
			itemComponent := TaskItem(TaskItemProperties{Task: updatedTask})
			_ = gossr.RenderHTTP(responseWriter, itemComponent)
			return
		}
		http.NotFound(responseWriter, request)
		return
	}

	http.Error(responseWriter, "Method not allowed", http.StatusMethodNotAllowed)
}

func setupRoutes() *http.ServeMux {
	globalTaskStore = NewTaskStore()
	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/tasks", handleTaskPage)
	serverMux.HandleFunc("/api/tasks", handleCreateTask)
	serverMux.HandleFunc("/api/tasks/search", handleSearchTasks)
	serverMux.HandleFunc("/api/tasks/", handleTaskMutation)
	return serverMux
}

func main() {
	serverMux := setupRoutes()
	fmt.Println("Server running on http://localhost:8080/tasks")
	_ = http.ListenAndServe(":8080", serverMux)
}
