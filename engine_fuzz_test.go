package gossr_test

import (
	"testing"

	"github.com/lemadane/gossr"
)

func FuzzRender(f *testing.F) {
	seeds := []string{
		"<div>${properties.Name}</div>",
		"<a href=\"${properties.Url}\">Link</a>",
		"<script>var x = \"${properties.Val}\";</script>",
		"<ul>${properties.Items.map(i => <li>${i}</li>)}</ul>",
		"<UserBadge name=\"${properties.Name}\" role=\"${properties.Role}\" />",
		"<!-- comment ${properties.Val} -->",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	type FuzzProps struct {
		Name  string
		Url   string
		Val   string
		Role  string
		Items []string
	}

	props := FuzzProps{
		Name:  "FuzzUser",
		Url:   "https://example.com/test",
		Val:   "FuzzValue",
		Role:  "Admin",
		Items: []string{"Item1", "Item2"},
	}

	f.Fuzz(func(t *testing.T, templateInput string) {
		comp := gossr.Render(templateInput, props)
		_ = comp.String()
	})
}

func FuzzSanitizeUrl(f *testing.F) {
	seeds := []string{
		"https://example.com",
		"javascript:alert(1)",
		"data:text/html;base64,PHNjcmlwdD4=",
		"/path/to/page#fragment?query=1",
		"java\x00script:alert(1)",
		"java&#115;cript:alert(1)",
		"mailto:test@example.com",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, urlInput string) {
		sanitized := gossr.SanitizeUrl(urlInput)
		_ = sanitized
	})
}
