package gossr_test

import (
	"strings"
	"testing"

	"github.com/lemadane/gossr"
)

type UserBadgeProperties struct {
	Name string
	Role string
}

func UserBadge(properties UserBadgeProperties) gossr.SSR {
	return gossr.Render(
		`<span class="badge ${properties.Role}">${properties.Name} (${properties.Role})</span>`,
		properties,
	)
}

type CardBoxProperties struct {
	Title    string
	Children gossr.SSR
}

func CardBox(properties CardBoxProperties) gossr.SSR {
	return gossr.Render(
		`<div class="card-box"><h3>${properties.Title}</h3><div class="body">${properties.Children}</div></div>`,
		properties,
	)
}

func TestSelfClosingCustomTag(testRunner *testing.T) {
	gossr.Register("UserBadge", UserBadge)

	componentInstance := gossr.Render(
		`<div><UserBadge name="Sarah Connor" role="Admin" /></div>`,
	)

	output := componentInstance.String()

	expected := `<div><span class="badge Admin">Sarah Connor (Admin)</span></div>`
	if output != expected {
		testRunner.Errorf("Expected self-closing custom tag to render %q, got %q", expected, output)
	}
}

func TestPairedCustomTagWithChildren(testRunner *testing.T) {
	gossr.Register("CardBox", CardBox)

	componentInstance := gossr.Render(
		`<CardBox title="Profile Header"><h1>User Profile Information</h1></CardBox>`,
	)

	output := componentInstance.String()

	expected := `<div class="card-box"><h3>Profile Header</h3><div class="body"><h1>User Profile Information</h1></div></div>`
	if output != expected {
		testRunner.Errorf("Expected paired custom tag to pass inner HTML to children %q, got %q", expected, output)
	}
}

func TestMapFactoryCustomTag(testRunner *testing.T) {
	gossr.Register("SimpleAlert", func(properties map[string]string) gossr.SSR {
		return gossr.Render(`<div class="alert">${properties.Message}</div>`, struct{ Message string }{Message: properties["message"]})
	})

	componentInstance := gossr.Render(`<SimpleAlert message="Operation succeeded" />`)
	output := componentInstance.String()

	if !strings.Contains(output, `<div class="alert">Operation succeeded</div>`) {
		testRunner.Errorf("Expected map factory tag to render output, got %q", output)
	}
}

func TestDynamicCustomTagAttributeEscaping(testRunner *testing.T) {
	gossr.Register("UserBadge", UserBadge)

	type PageProps struct {
		Name string
	}

	componentInstance := gossr.Render(
		`<div><UserBadge name="${properties.Name}" role="Admin" /></div>`,
		PageProps{Name: "Johnson & Johnson"},
	)

	output := componentInstance.String()

	expected := `<div><span class="badge Admin">Johnson &amp; Johnson (Admin)</span></div>`
	if output != expected {
		testRunner.Errorf("Expected dynamic custom tag prop to single-escape HTML entities %q, got %q", expected, output)
	}

	if strings.Contains(output, "&amp;amp;") {
		testRunner.Errorf("Detected double-escaping &amp;amp; in custom tag output: %q", output)
	}
}
