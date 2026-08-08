package gossr_test

import (
	"strings"
	"testing"

	"github.com/lemadane/gossr"
)

func BenchmarkSimpleRender(b *testing.B) {
	type Props struct {
		Title string
	}
	props := Props{Title: "GoSSR High Performance Engine"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comp := gossr.Render(`<h1>${properties.Title}</h1>`, props)
		_ = comp.String()
	}
}

func BenchmarkNestedPropertyResolution(b *testing.B) {
	type Geo struct {
		City string
	}
	type Address struct {
		Geo Geo
	}
	type Customer struct {
		Address Address
	}
	type Props struct {
		Customer Customer
	}
	props := Props{
		Customer: Customer{
			Address: Address{
				Geo: Geo{City: "San Francisco"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comp := gossr.Render(`<p>${properties.Customer.Address.Geo.City}</p>`, props)
		_ = comp.String()
	}
}

func BenchmarkMapIteration(b *testing.B) {
	type Task struct {
		Title string
		Done  bool
	}
	type Props struct {
		Tasks []Task
	}

	tasks := make([]Task, 50)
	for i := 0; i < 50; i++ {
		tasks[i] = Task{Title: "Benchmark task item", Done: i%2 == 0}
	}
	props := Props{Tasks: tasks}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comp := gossr.Render(`<ul>${properties.Tasks.map(t => <li class="${t.Done ? "completed" : "pending"}">${t.Title}</li>)}</ul>`, props)
		_ = comp.String()
	}
}

func BenchmarkUrlSanitization(b *testing.B) {
	urls := []string{
		"https://example.com/path?query=1",
		"javascript:alert(1)",
		"/relative/path#fragment",
		"data:text/html;base64,PHNjcmlwdD4=",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gossr.SanitizeUrl(urls[i%len(urls)])
	}
}

func BenchmarkCustomComponentRendering(b *testing.B) {
	type BadgeProps struct {
		Name string
		Role string
	}

	gossr.Register("BenchBadge", func(props BadgeProps) gossr.SSR {
		return gossr.Render(`<span class="badge ${properties.Role}">${properties.Name}</span>`, props)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comp := gossr.Render(`<div><BenchBadge name="Sarah Connor" role="Admin" /></div>`)
		_ = comp.String()
	}
}

func BenchmarkRenderToWriter(b *testing.B) {
	type Props struct {
		Title string
		Count int
	}
	props := Props{Title: "Stream Benchmark", Count: 42}

	var sb strings.Builder
	sb.Grow(256)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb.Reset()
		comp := gossr.Render(`<div class="box"><h2>${properties.Title}</h2><p>Count: ${properties.Count}</p></div>`, props)
		_ = comp.Render(&sb)
	}
}
