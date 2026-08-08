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
