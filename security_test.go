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
