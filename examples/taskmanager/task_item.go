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
