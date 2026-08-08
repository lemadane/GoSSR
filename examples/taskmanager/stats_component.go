package main

import "github.com/lemadane/gossr"

type TaskStatsProperties struct {
	Stats TaskStats
}

func TaskStatsComponent(properties TaskStatsProperties) gossr.SSR {
	return gossr.Render(`
		<div class="stats-grid">
			<div class="stat-card">
				<span class="stat-value">${properties.Stats.TotalCount}</span>
				<span class="stat-label">Total Tasks</span>
			</div>
			<div class="stat-card pending">
				<span class="stat-value">${properties.Stats.PendingCount}</span>
				<span class="stat-label">Pending</span>
			</div>
			<div class="stat-card completed">
				<span class="stat-value">${properties.Stats.CompletedCount}</span>
				<span class="stat-label">Completed</span>
			</div>
			<div class="stat-card high-priority">
				<span class="stat-value">${properties.Stats.HighPriority}</span>
				<span class="stat-label">High Priority</span>
			</div>
		</div>
	`, properties)
}
