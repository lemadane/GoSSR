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

			<div class="task-controls" x-data="{ activeFilter: 'all' }">
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
