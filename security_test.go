package gossr_test

import (
	"strings"
	"testing"

	"github.com/lemadane/gossr"
)

func TestSecurityContextRejectionScript(testRunner *testing.T) {
	type Props struct {
		Data string
	}

	comp := gossr.Render(`<script>var x = "${properties.Data}";</script>`, Props{Data: "test"})
	output := comp.String()

	if !strings.Contains(output, "Render Error") || !strings.Contains(output, "<script>") {
		testRunner.Errorf("Expected render error rejecting interpolation inside <script> block, got %q", output)
	}
}

func TestSecurityContextRejectionStyle(testRunner *testing.T) {
	type Props struct {
		Color string
	}

	comp := gossr.Render(`<style>body { color: ${properties.Color}; }</style>`, Props{Color: "red"})
	output := comp.String()

	if !strings.Contains(output, "Render Error") || !strings.Contains(output, "<style>") {
		testRunner.Errorf("Expected render error rejecting interpolation inside <style> block, got %q", output)
	}
}

func TestSecurityContextRejectionComment(testRunner *testing.T) {
	type Props struct {
		User string
	}

	comp := gossr.Render(`<!-- User: ${properties.User} -->`, Props{User: "admin"})
	output := comp.String()

	if !strings.Contains(output, "Render Error") || !strings.Contains(output, "comment") {
		testRunner.Errorf("Expected render error rejecting interpolation inside HTML comment, got %q", output)
	}
}

func TestSecurityContextRejectionInlineEventHandler(testRunner *testing.T) {
	type Props struct {
		Handler string
	}

	comp := gossr.Render(`<button onclick="${properties.Handler}">Click</button>`, Props{Handler: "alert(1)"})
	output := comp.String()

	if !strings.Contains(output, "Render Error") || !strings.Contains(output, "onclick") {
		testRunner.Errorf("Expected render error rejecting interpolation inside onclick attribute, got %q", output)
	}
}

func TestSecurityContextRejectionInlineStyleAttr(testRunner *testing.T) {
	type Props struct {
		Bg string
	}

	comp := gossr.Render(`<div style="background: ${properties.Bg}">Box</div>`, Props{Bg: "blue"})
	output := comp.String()

	if !strings.Contains(output, "Render Error") || !strings.Contains(output, "style") {
		testRunner.Errorf("Expected render error rejecting interpolation inside inline style attribute, got %q", output)
	}
}

func TestRawHtmlWrapper(testRunner *testing.T) {
	type Props struct {
		UnsafeText string
		Trusted    gossr.RawHtml
	}

	comp := gossr.Render(
		`<div><p>${properties.UnsafeText}</p><article>${properties.Trusted}</article></div>`,
		Props{
			UnsafeText: `<b>Not raw</b>`,
			Trusted:    gossr.Raw(`<b>Trusted Rich Text</b>`),
		},
	)

	output := comp.String()

	expectedEscaped := `&lt;b&gt;Not raw&lt;/b&gt;`
	if !strings.Contains(output, expectedEscaped) {
		testRunner.Errorf("Expected unsafe text to be HTML escaped %q, got %q", expectedEscaped, output)
	}

	if !strings.Contains(output, `<article><b>Trusted Rich Text</b></article>`) {
		testRunner.Errorf("Expected RawHtml to be output unescaped, got %q", output)
	}
}

func TestSafeUrlSanitizer(testRunner *testing.T) {
	type Props struct {
		ValidHttp   gossr.SafeUrl
		Relative    gossr.SafeUrl
		DangerousJS gossr.SafeUrl
		DataUri     gossr.SafeUrl
	}

	comp := gossr.Render(
		`<a href="${properties.ValidHttp}">HTTP</a>`+
			`<a href="${properties.Relative}">Rel</a>`+
			`<a href="${properties.DangerousJS}">JS</a>`+
			`<a href="${properties.DataUri}">Data</a>`,
		Props{
			ValidHttp:   gossr.URL("https://example.com/login?ref=1"),
			Relative:    gossr.URL("/dashboard#profile"),
			DangerousJS: gossr.URL("javascript:alert(1)"),
			DataUri:     gossr.URL("data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg=="),
		},
	)

	output := comp.String()

	if !strings.Contains(output, `href="https://example.com/login?ref=1"`) {
		testRunner.Errorf("Expected valid HTTP URL to be preserved, got %q", output)
	}

	if !strings.Contains(output, `href="/dashboard#profile"`) {
		testRunner.Errorf("Expected relative URL to be preserved, got %q", output)
	}

	if !strings.Contains(output, `href="about:blank"`) {
		testRunner.Errorf("Expected dangerous javascript: and data: URLs to be sanitized to about:blank, got %q", output)
	}
}

func TestNilPropertyHandling(testRunner *testing.T) {
	type Props struct {
		NilPointer *string
	}

	comp := gossr.Render(`<span>${properties.NilPointer}</span>`, Props{NilPointer: nil})
	output := comp.String()

	if output != `<span></span>` {
		testRunner.Errorf("Expected nil property to render empty string '<span></span>', got %q", output)
	}
}

func TestNormalStringInUrlAttributeAutoSanitized(testRunner *testing.T) {
	type Props struct {
		DangerousHref string
		DangerousSrc  string
		NormalUrl     string
		TrustedRaw    gossr.RawHtml
		TrustedSafe   gossr.SafeUrl
	}

	comp := gossr.Render(
		`<a href="${properties.DangerousHref}">Link</a>`+
			`<img src="${properties.DangerousSrc}" />`+
			`<a href="${properties.NormalUrl}">Normal</a>`+
			`<a href="${properties.TrustedRaw}">TrustedRaw</a>`+
			`<a href="${properties.TrustedSafe}">TrustedSafe</a>`,
		Props{
			DangerousHref: "javascript:alert(1)",
			DangerousSrc:  "data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==",
			NormalUrl:     "https://example.com/profile",
			TrustedRaw:    gossr.Raw("javascript:trustedFunction()"),
			TrustedSafe:   gossr.URL("https://example.com/trusted"),
		},
	)

	output := comp.String()

	if !strings.Contains(output, `href="about:blank"`) {
		testRunner.Errorf("Expected plain string javascript: URL in href attribute to be auto-sanitized to about:blank, got %q", output)
	}

	if !strings.Contains(output, `src="about:blank"`) {
		testRunner.Errorf("Expected plain string data: URL in src attribute to be auto-sanitized to about:blank, got %q", output)
	}

	if !strings.Contains(output, `href="https://example.com/profile"`) {
		testRunner.Errorf("Expected normal HTTPS URL in href attribute to be preserved, got %q", output)
	}

	if !strings.Contains(output, `href="https://example.com/trusted"`) {
		testRunner.Errorf("Expected SafeUrl wrapper to be allowed in URL attribute, got %q", output)
	}
}

func TestExecutableAttributeRejectionAlpineAndHTMX(testRunner *testing.T) {
	type Props struct {
		Val string
	}

	executableAttrTemplates := []string{
		`<div x-data="{ name: '${properties.Val}' }"></div>`,
		`<button @click="${properties.Val}">Btn</button>`,
		`<div x-init="${properties.Val}"></div>`,
		`<button hx-on:click="${properties.Val}">Btn</button>`,
		`<div hx-vals="${properties.Val}"></div>`,
		`<div :class="${properties.Val}"></div>`,
	}

	for _, tpl := range executableAttrTemplates {
		comp := gossr.Render(tpl, Props{Val: "test"})
		output := comp.String()

		if !strings.Contains(output, "Render Error") || !strings.Contains(output, "executable attribute") {
			testRunner.Errorf("Expected render error rejecting interpolation inside executable attribute for template %q, got %q", tpl, output)
		}
	}
}

type RecursiveTagProps struct {
	Depth int
}

func TestComponentRecursionDepthProtection(testRunner *testing.T) {
	gossr.Register("RecursiveLoopComponent", func(props RecursiveTagProps) gossr.SSR {
		return gossr.Render(`<RecursiveLoopComponent depth="${properties.Depth}" />`, props)
	})

	comp := gossr.Render(`<RecursiveLoopComponent depth="1" />`)
	output := comp.String()

	if !strings.Contains(output, "Render Error") || !strings.Contains(output, "recursion limit exceeded") {
		testRunner.Errorf("Expected render error indicating recursion limit exceeded, got %q", output)
	}
}

type SafeLinkComponentProps struct {
	Url  gossr.SafeUrl
	Body gossr.RawHtml
}

func TestCustomTagPropsReflectionNoPanic(testRunner *testing.T) {
	gossr.Register("SafeCustomLink", func(props SafeLinkComponentProps) gossr.SSR {
		return gossr.Render(`<a href="${properties.Url}">${properties.Body}</a>`, props)
	})

	comp := gossr.Render(`<SafeCustomLink url="javascript:alert(1)" body="<b>Click me</b>" />`)
	output := comp.String()

	if strings.Contains(output, "Render Error") {
		testRunner.Fatalf("Expected custom tag with SafeUrl and RawHtml props to render without error, got %q", output)
	}

	if !strings.Contains(output, `href="about:blank"`) {
		testRunner.Errorf("Expected SafeUrl field on custom tag props to sanitize dangerous URL to about:blank, got %q", output)
	}

	if !strings.Contains(output, `<b>Click me</b>`) {
		testRunner.Errorf("Expected RawHtml field on custom tag props to output unescaped HTML, got %q", output)
	}
}

func TestMapLambdaSecurityContextRejectionAndSanitization(testRunner *testing.T) {
	type Item struct {
		Payload string
		URL     string
		Text    string
	}
	type Props struct {
		Items []Item
	}

	items := []Item{
		{Payload: "alert(1)", URL: "javascript:alert(1)", Text: "<script>alert('x')</script>"},
	}
	props := Props{Items: items}

	rejectionTemplates := []string{
		`<div>${properties.Items.map(item => <button onclick="${item.Payload}">Click</button>)}</div>`,
		`<div>${properties.Items.map(item => <div x-data="{ value: '${item.Payload}' }"></div>)}</div>`,
		`<div>${properties.Items.map(item => <button @click="${item.Payload}">Btn</button>)}</div>`,
		`<div>${properties.Items.map(item => <button hx-on:click="${item.Payload}">Btn</button>)}</div>`,
		`<div>${properties.Items.map(item => <div style="${item.Payload}"></div>)}</div>`,
		`<div>${properties.Items.map(item => <script>${item.Payload}</script>)}</div>`,
		`<div>${properties.Items.map(item => <style>${item.Payload}</style>)}</div>`,
	}

	for _, tpl := range rejectionTemplates {
		comp := gossr.Render(tpl, props)
		output := comp.String()
		if !strings.Contains(output, "Render Error") || (!strings.Contains(output, "executable attribute") && !strings.Contains(output, "is not allowed inside")) {
			testRunner.Errorf("Expected render error rejecting map lambda interpolation for template %q, got %q", tpl, output)
		}
	}

	urlComp := gossr.Render(`<div>${properties.Items.map(item => <a href="${item.URL}">Link</a>)}</div>`, props)
	urlOutput := urlComp.String()
	if !strings.Contains(urlOutput, `href="about:blank"`) {
		testRunner.Errorf("Expected javascript: URL inside map lambda href to be auto-sanitized to about:blank, got %q", urlOutput)
	}

	textComp := gossr.Render(`<div>${properties.Items.map(item => <li>${item.Text}</li>)}</div>`, props)
	textOutput := textComp.String()
	if !strings.Contains(textOutput, `<li>&lt;script&gt;alert(&#39;x&#39;)&lt;/script&gt;</li>`) && !strings.Contains(textOutput, `<li>&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;</li>`) {
		testRunner.Errorf("Expected normal HTML text inside map lambda to be HTML escaped, got %q", textOutput)
	}
}
