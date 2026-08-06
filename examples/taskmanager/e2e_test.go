package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	if !strings.Contains(responseHtml, "Active Tasks Summary") {
		testRunner.Error("Response missing Card title 'Active Tasks Summary'")
	}

	// Check Alpine.js attributes presence
	if !strings.Contains(responseHtml, `x-data="{ collapsed: false }"`) {
		testRunner.Error("Response missing Alpine.js x-data attribute")
	}

	// Check HTMX deletion attributes
	if !strings.Contains(responseHtml, `hx-delete="/api/tasks/101"`) {
		testRunner.Error("Response missing HTMX hx-delete attribute for task 101")
	}
}

func TestE2EDeleteTaskEndpoint(testRunner *testing.T) {
	testServer := httptest.NewServer(setupRoutes())
	defer testServer.Close()

	request, createRequestError := http.NewRequest(http.MethodDelete, testServer.URL+"/api/tasks/102", nil)
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
