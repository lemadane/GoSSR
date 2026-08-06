package main

import "GoSSR/gossr"

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
