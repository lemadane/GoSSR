package gossr_test

import (
	"strings"
	"testing"

	"github.com/lemadane/gossr"
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

func TestHTMLEscapingXSS(testRunner *testing.T) {
	type Properties struct {
		Name     string
		Children gossr.SSR
	}

	childComponent := gossr.Render(`<span>Trusted Raw HTML</span>`, nil)

	component := gossr.Render(
		`<p>${properties.Name}</p><div>${properties.Children}</div>`,
		Properties{
			Name:     `<script>alert("x")</script>`,
			Children: childComponent,
		},
	)

	renderedOutput := component.String()

	expectedEscaped := `&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;`
	if !strings.Contains(renderedOutput, expectedEscaped) {
		testRunner.Errorf("Expected output to contain escaped XSS string %q, got %q", expectedEscaped, renderedOutput)
	}

	if !strings.Contains(renderedOutput, "<span>Trusted Raw HTML</span>") {
		testRunner.Errorf("Expected output to preserve trusted gossr.SSR HTML, got %q", renderedOutput)
	}
}

func TestMapDollarSignPreservation(testRunner *testing.T) {
	item := gossr.Render(`<li>$100 and $1</li>`, nil)

	type Properties struct {
		Items []gossr.SSR
	}

	component := gossr.Render(
		`<ul>${properties.Items.map(item => item)}</ul>`,
		Properties{
			Items: []gossr.SSR{item},
		},
	)

	renderedOutput := component.String()
	expectedOutput := `<ul><li>$100 and $1</li></ul>`

	if renderedOutput != expectedOutput {
		testRunner.Errorf("Expected dollar signs to be preserved verbatim %q, but got %q", expectedOutput, renderedOutput)
	}
}

func TestRealMapLambdaExecution(testRunner *testing.T) {
	type Task struct {
		Title string
		Done  bool
	}
	type Props struct {
		Tasks []Task
	}

	component := gossr.Render(
		`<ul>${properties.Tasks.map(t => <li class="${t.Done ? "completed" : "pending"}">${t.Title}</li>)}</ul>`,
		Props{
			Tasks: []Task{
				{Title: "Buy $5 milk", Done: true},
				{Title: "<script>alert('hack')</script>", Done: false},
			},
		},
	)

	renderedOutput := component.String()

	if !strings.Contains(renderedOutput, `<li class="completed">Buy $5 milk</li>`) {
		testRunner.Errorf("Expected output to contain completed item with $ preserved, got %q", renderedOutput)
	}

	expectedEscaped := `&lt;script&gt;alert(&#39;hack&#39;)&lt;/script&gt;`
	if !strings.Contains(renderedOutput, expectedEscaped) {
		testRunner.Errorf("Expected output to contain escaped title %q, got %q", expectedEscaped, renderedOutput)
	}

	if !strings.Contains(renderedOutput, `class="pending"`) {
		testRunner.Errorf("Expected output to contain pending class, got %q", renderedOutput)
	}
}

func TestArbitraryNestedPropertyDepth(testRunner *testing.T) {
	type Geo struct {
		City string
	}
	type Address struct {
		Geo Geo
	}
	type Customer struct {
		Address Address
	}
	type Properties struct {
		Customer Customer
	}

	component := gossr.Render(
		`<p>${properties.Customer.Address.Geo.City}</p>`,
		Properties{
			Customer: Customer{
				Address: Address{
					Geo: Geo{
						City: "San Francisco",
					},
				},
			},
		},
	)

	renderedOutput := component.String()
	expectedOutput := `<p>San Francisco</p>`

	if renderedOutput != expectedOutput {
		testRunner.Errorf("Expected 4-level nested property resolution %q, but got %q", expectedOutput, renderedOutput)
	}
}
