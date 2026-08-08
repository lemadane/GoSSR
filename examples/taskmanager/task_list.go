package main

import "github.com/lemadane/gossr"

type TaskListProperties struct {
	Title       string
	Tasks       []Task
	Stats       TaskStats
	SearchQuery string
	FilterTab   string
}

func TaskList(properties TaskListProperties) gossr.SSR {
	renderedItems := make([]gossr.SSR, len(properties.Tasks))
	for index, task := range properties.Tasks {
		renderedItems[index] = TaskItem(TaskItemProperties{Task: task})
	}

	statsComponent := TaskStatsComponent(TaskStatsProperties{Stats: properties.Stats})
	formComponent := TaskFormComponent(TaskFormProperties{})

	type TaskListRenderScope struct {
		Title          string
		StatsComponent gossr.SSR
		FormComponent  gossr.SSR
		Tasks          []gossr.SSR
		SearchQuery    string
		FilterTab      string
	}

	return gossr.Render(`
		<div id="task-list-container" class="page-container" data-filter="${properties.FilterTab}" x-data="{ currentFilter: $el.dataset.filter }">
			<header class="page-header">
				<h1>${properties.Title}</h1>
			</header>

			<!-- Task Statistics Component -->
			${properties.StatsComponent}

			<!-- Task Creation Form Component -->
			${properties.FormComponent}

			<!-- Live Search & Category Filter Controls -->
			<div class="search-filter-bar">
				<div class="search-input-wrapper">
					<input type="text" 
						   name="q" 
						   value="${properties.SearchQuery}"
						   placeholder="Search tasks..." 
						   class="input-search"
						   hx-get="/api/tasks/search" 
						   hx-trigger="keyup changed delay:250ms" 
						   hx-target="#task-list-container" 
						   hx-swap="outerHTML" />
				</div>

				<div class="filter-tabs">
					<button class="filter-tab" :class="{ 'active': currentFilter === 'all' }" @click="currentFilter = 'all'">All</button>
					<button class="filter-tab" :class="{ 'active': currentFilter === 'pending' }" @click="currentFilter = 'pending'">Pending</button>
					<button class="filter-tab" :class="{ 'active': currentFilter === 'completed' }" @click="currentFilter = 'completed'">Completed</button>
				</div>
			</div>

			<!-- Task List Container -->
			<div class="task-list-wrapper">
				<ul class="task-list">
					${properties.Tasks}
				</ul>
			</div>
		</div>
	`, TaskListRenderScope{
		Title:          properties.Title,
		StatsComponent: statsComponent,
		FormComponent:  formComponent,
		Tasks:          renderedItems,
		SearchQuery:    properties.SearchQuery,
		FilterTab:      properties.FilterTab,
	})
}
