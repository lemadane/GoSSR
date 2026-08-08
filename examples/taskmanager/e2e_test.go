package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/lemadane/gossr"
)

func TestE2ETaskPageEndpoint(testRunner *testing.T) {
	testServer := httptest.NewServer(setupRoutes())
	defer testServer.Close()

	response, getError := http.Get(testServer.URL + "/tasks")
	if getError != nil {
		testRunner.Fatalf("Failed to execute GET request: %v", getError)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		testRunner.Errorf("Expected status 200 OK, got %d", response.StatusCode)
	}

	contentType := response.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		testRunner.Errorf("Expected Content-Type text/html, got %q", contentType)
	}

	responseBodyBytes, readError := io.ReadAll(response.Body)
	if readError != nil {
		testRunner.Fatalf("Failed to read response body: %v", readError)
	}

	responseHtml := string(responseBodyBytes)

	// Check core HTML framework elements
	if !strings.Contains(responseHtml, "<!DOCTYPE html>") {
		testRunner.Error("Response missing DOCTYPE header")
	}

	// Check Card wrapper and Title
	if !strings.Contains(responseHtml, "Dashboard Summary") {
		testRunner.Error("Response missing Card title 'Dashboard Summary'")
	}

	// Check Alpine.js attributes presence
	if !strings.Contains(responseHtml, `x-data="{ expanded: false }"`) {
		testRunner.Error("Response missing Alpine.js x-data attribute")
	}

	// Check HTMX live search attribute
	if !strings.Contains(responseHtml, `hx-get="/api/tasks/search"`) {
		testRunner.Error("Response missing HTMX hx-get search attribute")
	}
}

func TestE2ECreateTaskEndpoint(testRunner *testing.T) {
	testServer := httptest.NewServer(setupRoutes())
	defer testServer.Close()

	formData := url.Values{}
	formData.Set("title", "Write comprehensive E2E tests")
	formData.Set("priority", "HIGH")

	response, err := http.PostForm(testServer.URL+"/api/tasks", formData)
	if err != nil {
		testRunner.Fatalf("Failed POST request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		testRunner.Errorf("Expected status 200 OK, got %d", response.StatusCode)
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		testRunner.Fatalf("Failed reading body: %v", err)
	}

	responseHtml := string(bodyBytes)
	if !strings.Contains(responseHtml, "Write comprehensive E2E tests") {
		testRunner.Errorf("Expected body to contain newly created task title, got %q", responseHtml)
	}
	if !strings.Contains(responseHtml, `priority-badge HIGH`) {
		testRunner.Errorf("Expected body to contain HIGH priority badge, got %q", responseHtml)
	}
}

func TestE2EToggleTaskEndpoint(testRunner *testing.T) {
	testServer := httptest.NewServer(setupRoutes())
	defer testServer.Close()

	// Initial tasks include task 101, 102, 103, 104
	request, err := http.NewRequest(http.MethodPut, testServer.URL+"/api/tasks/102/toggle", nil)
	if err != nil {
		testRunner.Fatalf("Failed creating PUT request: %v", err)
	}

	httpClient := &http.Client{}
	response, err := httpClient.Do(request)
	if err != nil {
		testRunner.Fatalf("Failed executing PUT request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		testRunner.Errorf("Expected status 200 OK for toggle endpoint, got %d", response.StatusCode)
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		testRunner.Fatalf("Failed reading body: %v", err)
	}

	responseHtml := string(bodyBytes)
	if !strings.Contains(responseHtml, `class="task-item completed`) {
		testRunner.Errorf("Expected toggled task item to render completed class, got %q", responseHtml)
	}
}

func TestE2EDeleteTaskEndpoint(testRunner *testing.T) {
	testServer := httptest.NewServer(setupRoutes())
	defer testServer.Close()

	request, createRequestError := http.NewRequest(http.MethodDelete, testServer.URL+"/api/tasks/103", nil)
	if createRequestError != nil {
		testRunner.Fatalf("Failed to create DELETE request: %v", createRequestError)
	}

	httpClient := &http.Client{}
	response, executeRequestError := httpClient.Do(request)
	if executeRequestError != nil {
		testRunner.Fatalf("Failed to execute DELETE request: %v", executeRequestError)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		testRunner.Errorf("Expected status 200 OK for DELETE endpoint, got %d", response.StatusCode)
	}
}

func TestE2ESearchEndpoint(testRunner *testing.T) {
	testServer := httptest.NewServer(setupRoutes())
	defer testServer.Close()

	response, err := http.Get(testServer.URL + "/api/tasks/search?q=Alpine")
	if err != nil {
		testRunner.Fatalf("Failed GET search request: %v", err)
	}
	defer response.Body.Close()

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		testRunner.Fatalf("Failed reading body: %v", err)
	}

	responseHtml := string(bodyBytes)
	if !strings.Contains(responseHtml, "Attach Alpine.js client dropdown &amp; modal state") && !strings.Contains(responseHtml, "Attach Alpine.js client dropdown & modal state") {
		testRunner.Errorf("Expected search response to contain Alpine task, got %q", responseHtml)
	}
}

func TestE2ESecurityAndDollarPreservation(testRunner *testing.T) {
	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/adversarial-tasks", func(responseWriter http.ResponseWriter, request *http.Request) {
		adversarialTasks := []Task{
			{ID: "999", Title: `Pay $100 and $1 for <script>alert("xss")</script>`, Completed: false, Priority: "HIGH"},
		}

		taskListComponent := TaskList(TaskListProperties{
			Title:    "Adversarial Tasks",
			Tasks:    adversarialTasks,
			Stats:    TaskStats{TotalCount: 1},
		})

		pageHtml := fmt.Sprintf("<html><body>%s</body></html>", taskListComponent.String())
		responseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
		responseWriter.Write([]byte(pageHtml))
	})

	testServer := httptest.NewServer(serverMux)
	defer testServer.Close()

	response, err := http.Get(testServer.URL + "/adversarial-tasks")
	if err != nil {
		testRunner.Fatalf("Failed to fetch adversarial task endpoint: %v", err)
	}
	defer response.Body.Close()

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		testRunner.Fatalf("Failed to read response body: %v", err)
	}

	htmlOutput := string(bodyBytes)

	// 1. Verify XSS script is escaped properly
	expectedEscapedScript := `&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;`
	if !strings.Contains(htmlOutput, expectedEscapedScript) && !strings.Contains(htmlOutput, `&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;`) {
		testRunner.Errorf("Expected XSS payload to be escaped in E2E response, got: %s", htmlOutput)
	}

	// 2. Verify dollar amounts $100 and $1 are preserved without regex corruption
	if !strings.Contains(htmlOutput, "$100 and $1") {
		testRunner.Errorf("Expected '$100 and $1' to be preserved verbatim in E2E response, got: %s", htmlOutput)
	}
}

func TestE2EHTTPHandlerAndCustomTags(testRunner *testing.T) {
	type TaskCardProperties struct {
		Title string
	}

	gossr.Register("E2ETaskCard", func(properties TaskCardProperties) gossr.SSR {
		return gossr.Render(`<div class="e2e-card"><h3>${properties.Title}</h3></div>`, properties)
	})

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/custom-tag-page", gossr.Handler(func(request *http.Request) gossr.SSR {
		return gossr.Render(`<div><E2ETaskCard title="Registered Component" /></div>`)
	}))

	testServer := httptest.NewServer(serverMux)
	defer testServer.Close()

	response, err := http.Get(testServer.URL + "/custom-tag-page")
	if err != nil {
		testRunner.Fatalf("Failed GET request: %v", err)
	}
	defer response.Body.Close()

	if response.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		testRunner.Errorf("Expected Content-Type header 'text/html; charset=utf-8', got %q", response.Header.Get("Content-Type"))
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		testRunner.Fatalf("Failed reading body: %v", err)
	}

	body := string(bodyBytes)
	if !strings.Contains(body, `<div class="e2e-card"><h3>Registered Component</h3></div>`) {
		testRunner.Errorf("Expected body to contain rendered custom tag output, got %q", body)
	}
}

func TestE2EFormControlsCheckboxesRadio(testRunner *testing.T) {
	type FormProperties struct {
		Name      string
		Subscribe bool
		Role      string
	}

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/form", gossr.Handler(func(request *http.Request) gossr.SSR {
		return gossr.Render(
			`<form hx-post="/form">`+
				`<input name="name" value="${properties.Name}" />`+
				`<input type="checkbox" name="subscribe" ${properties.Subscribe ? "checked" : ""} />`+
				`<input type="radio" name="role" value="ADMIN" ${properties.Role == "ADMIN" ? "checked" : ""} />`+
				`<input type="radio" name="role" value="USER" ${properties.Role == "USER" ? "checked" : ""} />`+
				`</form>`,
			FormProperties{
				Name:      `John "Developer" Doe`,
				Subscribe: true,
				Role:      "ADMIN",
			},
		)
	}))

	testServer := httptest.NewServer(serverMux)
	defer testServer.Close()

	response, err := http.Get(testServer.URL + "/form")
	if err != nil {
		testRunner.Fatalf("Failed GET request: %v", err)
	}
	defer response.Body.Close()

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		testRunner.Fatalf("Failed reading body: %v", err)
	}

	body := string(bodyBytes)

	if !strings.Contains(body, `value="John &quot;Developer&quot; Doe"`) && !strings.Contains(body, `value="John &#34;Developer&#34; Doe"`) {
		testRunner.Errorf("Expected pre-filled name input quote escaping, got %q", body)
	}

	if !strings.Contains(body, `<input type="checkbox" name="subscribe" checked />`) {
		testRunner.Errorf("Expected checkbox checked attribute, got %q", body)
	}

	if !strings.Contains(body, `<input type="radio" name="role" value="ADMIN" checked />`) {
		testRunner.Errorf("Expected ADMIN radio button checked attribute, got %q", body)
	}
}
