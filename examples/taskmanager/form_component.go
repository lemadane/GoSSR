package main

import "github.com/lemadane/gossr"

type TaskFormProperties struct {
	ErrorMessage string
	SuccessMessage string
}

func TaskFormComponent(properties TaskFormProperties) gossr.SSR {
	return gossr.Render(`
		<div class="task-form-wrapper" x-data="{ expanded: false }">
			<div class="form-header">
				<h3>Create New Task</h3>
				<button class="button-toggle" @click="expanded = !expanded">
					<span x-show="!expanded">+ Add Task</span>
					<span x-show="expanded">&times; Close</span>
				</button>
			</div>

			<div x-show="expanded" x-transition class="form-content">
				${properties.ErrorMessage != "" ? "<p class=\"error-banner\">" : ""}${properties.ErrorMessage}${properties.ErrorMessage != "" ? "</p>" : ""}
				${properties.SuccessMessage != "" ? "<p class=\"success-banner\">" : ""}${properties.SuccessMessage}${properties.SuccessMessage != "" ? "</p>" : ""}

				<form hx-post="/api/tasks" hx-target="#task-list-container" hx-swap="outerHTML" @submit="expanded = false">
					<div class="form-group">
						<label for="task-title-input">Task Description</label>
						<input id="task-title-input" type="text" name="title" placeholder="e.g., Implement OAuth2 integration" required class="input-text" />
					</div>

					<div class="form-group">
						<label>Priority Level</label>
						<div class="radio-group">
							<label class="radio-label">
								<input type="radio" name="priority" value="HIGH" /> High
							</label>
							<label class="radio-label">
								<input type="radio" name="priority" value="MEDIUM" checked /> Medium
							</label>
							<label class="radio-label">
								<input type="radio" name="priority" value="LOW" /> Low
							</label>
						</div>
					</div>

					<div class="form-actions">
						<button type="submit" class="button-submit">Create Task</button>
					</div>
				</form>
			</div>
		</div>
	`, properties)
}
