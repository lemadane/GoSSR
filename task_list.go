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

			<div class="task-controls" x-data="{ activeFilter: 'all' }">
				<ul class="task-list">
					${properties.TaskList.map(singleTask => TaskItem(TaskItemProperties{Task: singleTask}))}
				</ul>
			</div>
		</main>
	`, properties)
}
