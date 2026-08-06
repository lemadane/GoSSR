package gossr_test

import (
	"strings"
	"testing"

	"GoSSR/gossr"
)

func TestSimplePropertySubstitution(testRunner *testing.T) {
	type SimpleProperties struct {
		Heading string
	}

	component := gossr.Render(`<h1>${properties.Heading}</h1>`, SimpleProperties{Heading: "Welcome GoSSR"})
	renderedOutput := component.String()

	expectedContent := "<h1>Welcome GoSSR</h1>"
	if renderedOutput != expectedContent {
		testRunner.Errorf("Expected %q, but got %q", expectedContent, renderedOutput)
	}
}

func TestNestedPropertySubstitution(testRunner *testing.T) {
	type Product struct {
		ID   string
		Name string
	}
	type ProductProperties struct {
		Item Product
	}

	component := gossr.Render(`<div id="prod-${properties.Item.ID}">${properties.Item.Name}</div>`, ProductProperties{
		Item: Product{ID: "505", Name: "Go Laptop"},
	})
	renderedOutput := component.String()

	if !strings.Contains(renderedOutput, "prod-505") {
		testRunner.Errorf("Expected output to contain 'prod-505', got %q", renderedOutput)
	}
	if !strings.Contains(renderedOutput, "Go Laptop") {
		testRunner.Errorf("Expected output to contain product name, got %q", renderedOutput)
	}
}

func TestChildComponentRendering(testRunner *testing.T) {
	type ChildProperties struct {
		Message string
	}
	childComponent := gossr.Render(`<span>${properties.Message}</span>`, ChildProperties{Message: "Child Element"})

	type WrapperProperties struct {
		Title    string
		Children gossr.SSR
	}

	parentComponent := gossr.Render(`<div class="wrapper"><h3>${properties.Title}</h3>${properties.Children}</div>`, WrapperProperties{
		Title:    "Parent Wrapper",
		Children: childComponent,
	})

	renderedOutput := parentComponent.String()

	if !strings.Contains(renderedOutput, "Parent Wrapper") {
		testRunner.Errorf("Expected output to contain 'Parent Wrapper', got %q", renderedOutput)
	}
	if !strings.Contains(renderedOutput, "<span>Child Element</span>") {
		testRunner.Errorf("Expected output to contain child element string, got %q", renderedOutput)
	}
}

func TestTernaryExpression(testRunner *testing.T) {
	type FeatureProperties struct {
		Enabled bool
	}

	enabledComponent := gossr.Render(`<div>Status: ${properties.Enabled ? "Active" : "Inactive"}</div>`, FeatureProperties{Enabled: true})
	disabledComponent := gossr.Render(`<div>Status: ${properties.Enabled ? "Active" : "Inactive"}</div>`, FeatureProperties{Enabled: false})

	if !strings.Contains(enabledComponent.String(), "Active") {
		testRunner.Errorf("Expected enabled component to show 'Active', got %q", enabledComponent.String())
	}
	if !strings.Contains(disabledComponent.String(), "Inactive") {
		testRunner.Errorf("Expected disabled component to show 'Inactive', got %q", disabledComponent.String())
	}
}

func TestGenericSliceMapping(testRunner *testing.T) {
	itemOne := gossr.Render(`<li>Item 1</li>`, nil)
	itemTwo := gossr.Render(`<li>Item 2</li>`, nil)

	type ListProperties struct {
		Items []gossr.SSR
	}

	listComponent := gossr.Render(`<ul>${properties.Items.map(item => item)}</ul>`, ListProperties{
		Items: []gossr.SSR{itemOne, itemTwo},
	})

	renderedOutput := listComponent.String()

	if !strings.Contains(renderedOutput, "<li>Item 1</li><li>Item 2</li>") {
		testRunner.Errorf("Expected rendered generic slice to contain list items, got %q", renderedOutput)
	}
}
