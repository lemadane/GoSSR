package gossr_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lemadane/gossr"
)

func TestExportedHTMLEntityHelpers(testRunner *testing.T) {
	raw := `<script>alert("1 & 2")</script>`
	escaped := gossr.EscapeHTML(raw)

	expectedEscaped := `&lt;script&gt;alert(&#34;1 &amp; 2&#34;)&lt;/script&gt;`
	if escaped != expectedEscaped && escaped != `&lt;script&gt;alert(&quot;1 &amp; 2&quot;)&lt;/script&gt;` {
		testRunner.Errorf("Unexpected EscapeHTML output %q", escaped)
	}

	unescaped := gossr.UnescapeHTML(escaped)
	if unescaped != raw {
		testRunner.Errorf("Expected UnescapeHTML to restore raw string %q, got %q", raw, unescaped)
	}
}

func TestRenderHTTP(testRunner *testing.T) {
	type Properties struct {
		Message string
	}
	componentInstance := gossr.Render(`<h1>${properties.Message}</h1>`, Properties{Message: "HTTP Test"})

	responseRecorder := httptest.NewRecorder()
	renderError := gossr.RenderHTTP(responseRecorder, componentInstance)
	if renderError != nil {
		testRunner.Fatalf("RenderHTTP failed: %v", renderError)
	}

	if responseRecorder.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		testRunner.Errorf("Expected Content-Type header 'text/html; charset=utf-8', got %q", responseRecorder.Header().Get("Content-Type"))
	}

	if responseRecorder.Body.String() != "<h1>HTTP Test</h1>" {
		testRunner.Errorf("Expected body '<h1>HTTP Test</h1>', got %q", responseRecorder.Body.String())
	}
}

func TestHTTPHandlerWrapper(testRunner *testing.T) {
	type Properties struct {
		Path string
	}

	handler := gossr.Handler(func(request *http.Request) gossr.SSR {
		return gossr.Render(`<p>Path: ${properties.Path}</p>`, Properties{Path: request.URL.Path})
	})

	httpRequest := httptest.NewRequest("GET", "/hello/world", nil)
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, httpRequest)

	if responseRecorder.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		testRunner.Errorf("Expected Content-Type header 'text/html; charset=utf-8', got %q", responseRecorder.Header().Get("Content-Type"))
	}

	if responseRecorder.Body.String() != "<p>Path: /hello/world</p>" {
		testRunner.Errorf("Expected body '<p>Path: /hello/world</p>', got %q", responseRecorder.Body.String())
	}
}

func TestCustomTagAttributeTypoRejection(testRunner *testing.T) {
	gossr.Register("StrictUserCard", UserBadge)

	componentInstance := gossr.Render(`<StrictUserCard nme="Sarah" role="Admin" />`)
	output := componentInstance.String()

	if !strings.Contains(output, "Render Error") || !strings.Contains(output, "unrecognized attribute 'nme'") {
		testRunner.Errorf("Expected render error rejecting typo attribute 'nme', got %q", output)
	}
}
