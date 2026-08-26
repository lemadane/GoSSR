package gossr_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lemadane/gossr"
)

func TestFrameworkVersion(testRunner *testing.T) {
	if gossr.Version != "0.3.0" {
		testRunner.Fatalf("Expected framework version 0.3.0, got %q", gossr.Version)
	}
}

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

func TestAllDollarSignVariationsInMap(testRunner *testing.T) {
	type Product struct {
		Name  string
		Price string
	}
	type Props struct {
		Products []Product
	}

	comp := gossr.Render(
		`<ul>${properties.Products.map(p => <li>${p.Name}: ${p.Price}</li>)}</ul>`,
		Props{
			Products: []Product{
				{Name: "Item $1", Price: "$100 and $1"},
				{Name: "${product}", Price: "Price: $150"},
			},
		},
	)

	output := comp.String()
	if !strings.Contains(output, "<li>Item $1: $100 and $1</li>") {
		testRunner.Errorf("Expected literal $1, $100, $1 preservation in map output, got %q", output)
	}
	if !strings.Contains(output, "<li>${product}: Price: $150</li>") {
		testRunner.Errorf("Expected literal ${product} and Price: $150 in map output, got %q", output)
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

func TestMapLambdaWithCustomComponentTag(testRunner *testing.T) {
	type User struct {
		Name string
		Role string
	}
	type Props struct {
		Users []User
	}

	gossr.Register("LambdaBadge", UserBadge)

	comp := gossr.Render(
		`<div class="team">${properties.Users.map(u => <LambdaBadge name="${u.Name}" role="${u.Role}" />)}</div>`,
		Props{
			Users: []User{
				{Name: "Sarah Connor", Role: "Admin"},
				{Name: "Alex Mercer", Role: "Developer"},
			},
		},
	)

	output := comp.String()
	expected := `<div class="team"><span class="badge Admin">Sarah Connor (Admin)</span><span class="badge Developer">Alex Mercer (Developer)</span></div>`

	if output != expected {
		testRunner.Errorf("Expected map lambda to render custom tags %q, got %q", expected, output)
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

	type SimpleAddress struct {
		City string
	}
	type SimpleCustomer struct {
		Address SimpleAddress
	}
	type ThreeLevelProps struct {
		Customer SimpleCustomer
	}

	threeLevelComp := gossr.Render(
		`<p>${properties.Customer.Address.City}</p>`,
		ThreeLevelProps{
			Customer: SimpleCustomer{
				Address: SimpleAddress{
					City: "Manila",
				},
			},
		},
	)

	if threeLevelComp.String() != `<p>Manila</p>` {
		testRunner.Errorf("Expected 3-level nested property resolution '<p>Manila</p>', got %q", threeLevelComp.String())
	}
}

func TestHandlerErrorPropagation(testRunner *testing.T) {
	handler := gossr.Handler(func(req *http.Request) gossr.SSR {
		return gossr.Render(`<script>${properties.Val}</script>`, struct{ Val string }{Val: "unsafe"})
	})

	req := httptest.NewRequest(http.MethodGet, "/error-test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		testRunner.Errorf("Expected HTTP status 500 Internal Server Error when rendering fails, got %d", rec.Code)
	}
}

func TestStrictModeUnresolvedPropertyError(testRunner *testing.T) {
	gossr.Strict(true)
	defer gossr.Strict(false)

	type User struct {
		Name string
	}

	comp := gossr.Render(`<p>${properties.Custmer.Name}</p>`, User{Name: "Sarah"})
	output := comp.String()

	if !strings.Contains(output, "Render Error") || !strings.Contains(output, "unresolved property path") {
		testRunner.Errorf("Expected strict mode render error for unresolved property path typo, got %q", output)
	}
}

func TestCompiledTemplateAndMustCompile(testRunner *testing.T) {
	tpl := gossr.MustCompile(`<div>${properties.Name}</div>`)

	type User struct {
		Name string
	}

	comp := tpl.Bind(User{Name: "Sarah"})
	if comp.String() != `<div>Sarah</div>` {
		testRunner.Errorf("Expected compiled template Bind output '<div>Sarah</div>', got %q", comp.String())
	}

	var sb strings.Builder
	if err := tpl.Render(&sb, User{Name: "Alex"}); err != nil {
		testRunner.Fatalf("Unexpected compiled template Render error: %v", err)
	}
	if sb.String() != `<div>Alex</div>` {
		testRunner.Errorf("Expected compiled template Render output '<div>Alex</div>', got %q", sb.String())
	}

	_, err := gossr.Compile(`<script>${properties.Val}</script>`)
	if err == nil {
		testRunner.Error("Expected Compile to fail on dangerous script block template, got nil error")
	}

	defer func() {
		if r := recover(); r == nil {
			testRunner.Error("Expected MustCompile to panic on dangerous template, got no panic")
		}
	}()
	_ = gossr.MustCompile(`<script>${properties.Val}</script>`)
}

func TestRawHtmlInsideAttributeIsEscaped(testRunner *testing.T) {
	comp := gossr.Render(`<div id="${properties.Raw}"></div>`, struct {
		Raw gossr.RawHtml
	}{
		Raw: gossr.Raw(`" onclick="alert(1)`),
	})

	output := comp.String()
	if strings.Contains(output, `id="" onclick="alert(1)"`) || (!strings.Contains(output, `&#34;`) && !strings.Contains(output, `&quot;`)) {
		testRunner.Errorf("Expected RawHtml inside attribute context to be double-quote escaped, got %q", output)
	}
}

func TestNestedChildComponentErrorPropagation(testRunner *testing.T) {
	child := gossr.Render(`<script>${properties.Value}</script>`, struct {
		Value string
	}{
		Value: "unsafe",
	})

	parent := gossr.Render(`<div>${properties.Child}</div>`, struct {
		Child gossr.SSR
	}{
		Child: child,
	})

	var sb strings.Builder
	err := parent.Render(&sb)
	if err == nil {
		testRunner.Error("Expected parent.Render to fail when child component returns render error, got nil error")
	}
}

func TestNonStringKeyedMapNoPanic(testRunner *testing.T) {
	type IntMapProps struct {
		Map map[int]string
	}
	intComp := gossr.Render(`<p>${properties.Map.42}</p>`, IntMapProps{Map: map[int]string{42: "Answer"}})
	if intComp.String() != `<p>Answer</p>` {
		testRunner.Errorf("Expected int-keyed map resolution '<p>Answer</p>', got %q", intComp.String())
	}

	type StructKeyProps struct {
		Map map[struct{}]string
	}
	structComp := gossr.Render(`<p>${properties.Map.key}</p>`, StructKeyProps{Map: map[struct{}]string{{}: "Val"}})
	_ = structComp.String() // Must NOT panic!

	type PropertyName string
	type NamedStringMapProps struct {
		Map map[PropertyName]string
	}
	namedComp := gossr.Render(`<p>${properties.Map.key}</p>`, NamedStringMapProps{Map: map[PropertyName]string{"key": "value"}})
	if namedComp.String() != `<p>value</p>` {
		testRunner.Errorf("Expected named-string map key resolution '<p>value</p>', got %q", namedComp.String())
	}
}

func TestRawHtmlInUrlAttributeIsSanitized(testRunner *testing.T) {
	comp := gossr.Render(`<a href="${properties.Link}">Link</a>`, struct {
		Link gossr.RawHtml
	}{
		Link: gossr.Raw("javascript:alert(1)"),
	})

	output := comp.String()
	if !strings.Contains(output, `href="about:blank"`) {
		testRunner.Errorf("Expected RawHtml containing javascript: in href to sanitize to about:blank, got %q", output)
	}
}

func TestCompiledTemplateParity(testRunner *testing.T) {
	gossr.Register("TestBadge", func(props struct{ Name string }) gossr.SSR {
		return gossr.Render(`<span class="badge">${properties.Name}</span>`, props)
	})

	type Task struct {
		Name   string
		Active bool
	}

	type User struct {
		Name     string
		Role     string
		Active   bool
		Items    []Task
		RawText  gossr.RawHtml
		SafeLink gossr.SafeUrl
	}

	userData := User{
		Name:     "Sarah",
		Role:     "ADMIN",
		Active:   true,
		Items:    []Task{{Name: "Task 1", Active: true}, {Name: "Task 2", Active: false}},
		RawText:  gossr.Raw("<b>Trusted</b>"),
		SafeLink: gossr.URL("https://example.com"),
	}

	templates := []string{
		`<div>${properties.Name}</div>`,
		`<div class="${properties.Active ? "active" : "inactive"}">${properties.Name}</div>`,
		`<div>${properties.Role == "ADMIN" ? "checked" : "unchecked"}</div>`,
		`<ul>${properties.Items.map(item => <li class="${item.Active ? "active" : "inactive"}">${item.Name}</li>)}</ul>`,
		`<p>${properties.RawText}</p>`,
		`<a href="${properties.SafeLink}">Link</a>`,
		`<TestBadge name="Alex" />`,
	}

	for idx, tplStr := range templates {
		normalResult := gossr.Render(tplStr, userData).String()
		compiledResult := gossr.MustCompile(tplStr).Bind(userData).String()

		if normalResult != compiledResult {
			testRunner.Errorf("Template %d parity mismatch!\nNormal Output:   %q\nCompiled Output: %q", idx, normalResult, compiledResult)
		}
	}
}
