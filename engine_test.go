package main

import (
	"strings"
	"testing"
)

func TestSimplePropertySubstitution(testRunner *testing.T) {
	type SimpleProperties struct {
		Heading string
	}

	component := Render(`<h1>${properties.Heading}</h1>`, SimpleProperties{Heading: "Welcome GoSSR"})
	renderedOutput := component.String()

	expectedContent := "<h1>Welcome GoSSR</h1>"
	if renderedOutput != expectedContent {
		testRunner.Errorf("Expected %q, but got %q", expectedContent, renderedOutput)
	}
}

func TestNestedPropertySubstitution(testRunner *testing.T) {
	singleTask := Task{
		ID:        "999",
		Title:     "Build Unit Test Engine",
		Completed: true,
	}

	component := TaskItem(TaskItemProperties{Task: singleTask})
	renderedOutput := component.String()

	if !strings.Contains(renderedOutput, "task-999") {
		testRunner.Errorf("Expected output to contain 'task-999', got %q", renderedOutput)
	}
	if !strings.Contains(renderedOutput, "Build Unit Test Engine") {
		testRunner.Errorf("Expected output to contain task title, got %q", renderedOutput)
	}
}

func TestChildComponentRendering(testRunner *testing.T) {
	type ChildProperties struct {
		Message string
	}
	childComponent := Render(`<span>${properties.Message}</span>`, ChildProperties{Message: "Child Element"})

	cardComponent := Card(CardProperties{
		Title:    "Parent Wrapper",
		Children: childComponent,
	})

	renderedOutput := cardComponent.String()

	if !strings.Contains(renderedOutput, "Parent Wrapper") {
		testRunner.Errorf("Expected output to contain 'Parent Wrapper', got %q", renderedOutput)
	}
	if !strings.Contains(renderedOutput, "<span>Child Element</span>") {
		testRunner.Errorf("Expected output to contain child element string, got %q", renderedOutput)
	}
}

func TestTernaryExpression(testRunner *testing.T) {
	completedTask := Task{ID: "1", Title: "Finished Task", Completed: true}
	pendingTask := Task{ID: "2", Title: "Unfinished Task", Completed: false}

	completedComponent := TaskItem(TaskItemProperties{Task: completedTask})
	pendingComponent := TaskItem(TaskItemProperties{Task: pendingTask})

	if !strings.Contains(completedComponent.String(), "Completed") {
		testRunner.Errorf("Expected completed task to show 'Completed', got %q", completedComponent.String())
	}
	if !strings.Contains(pendingComponent.String(), "Pending") {
		testRunner.Errorf("Expected pending task to show 'Pending', got %q", pendingComponent.String())
	}
}

func TestSliceMapping(testRunner *testing.T) {
	sampleTasks := []Task{
		{ID: "1", Title: "Task One", Completed: true},
		{ID: "2", Title: "Task Two", Completed: false},
	}

	listComponent := TaskList(TaskListProperties{
		Title:    "My Tasks",
		TaskList: sampleTasks,
	})

	renderedOutput := listComponent.String()

	if !strings.Contains(renderedOutput, "task-1") || !strings.Contains(renderedOutput, "task-2") {
		testRunner.Errorf("Expected rendered slice to contain task-1 and task-2, got %q", renderedOutput)
	}
}
