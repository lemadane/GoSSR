package gossr_test

import (
	"strings"
	"testing"

	"github.com/lemadane/gossr"
)

func TestControlFlowIfElse(testRunner *testing.T) {
	type Props struct {
		State string
		Name  string
	}

	component := gossr.Render(
		`@if State == "ready" {<p>${properties.Name}</p>} @elseif State == "pending" {<span>Pending</span>} @else {<em>Stopped</em>}`,
		Props{State: "ready", Name: "Launch"},
	)

	output := component.String()
	if !strings.Contains(output, `<p>Launch</p>`) {
		testRunner.Fatalf("Expected @if branch to render, got %q", output)
	}

	component = gossr.Render(
		`@if State == "ready" {<p>${properties.Name}</p>} @elseif State == "pending" {<span>Pending</span>} @else {<em>Stopped</em>}`,
		Props{State: "pending", Name: "Launch"},
	)
	output = component.String()
	if !strings.Contains(output, `<span>Pending</span>`) {
		testRunner.Fatalf("Expected @elseif branch to render, got %q", output)
	}

	component = gossr.Render(
		`@if State == "ready" {<p>${properties.Name}</p>} @elseif State == "pending" {<span>Pending</span>} @else {<em>Stopped</em>}`,
		Props{State: "halted", Name: "Launch"},
	)
	output = component.String()
	if !strings.Contains(output, `<em>Stopped</em>`) {
		testRunner.Fatalf("Expected @else branch to render, got %q", output)
	}
}

func TestControlFlowForElseBreakAndContinue(testRunner *testing.T) {
	type Props struct {
		Items []string
	}

	component := gossr.Render(
		`<ul>@for index, value in Items {@if value == "skip" { @continue }<li>${index}:${value}</li>@if value == "stop" { @break }}</ul>`,
		Props{Items: []string{"skip", "one", "stop", "two"}},
	)

	output := component.String()
	if strings.Contains(output, "skip") || strings.Contains(output, "two") {
		testRunner.Fatalf("Expected continue/break to stop skipped items, got %q", output)
	}
	if !strings.Contains(output, `<li>1:one</li>`) || !strings.Contains(output, `<li>2:stop</li>`) {
		testRunner.Fatalf("Expected loop body to render remaining items before break, got %q", output)
	}

	emptyComponent := gossr.Render(
		`<ul>@for index, value in Items {<li>${index}:${value}</li>} @else {<li>empty</li>}</ul>`,
		Props{Items: nil},
	)

	if !strings.Contains(emptyComponent.String(), `<li>empty</li>`) {
		testRunner.Fatalf("Expected @for @else fallback for empty collection, got %q", emptyComponent.String())
	}

	// Test single variable syntax: @for value in Items
	singleVarComponent := gossr.Render(
		`<ul>@for item in Items {<li>${item}</li>}</ul>`,
		Props{Items: []string{"a", "b"}},
	)
	if !strings.Contains(singleVarComponent.String(), `<li>a</li><li>b</li>`) {
		testRunner.Fatalf("Expected single variable @for loop to render items, got %q", singleVarComponent.String())
	}

	// Test that unsupported keywords ('range' and 'from') are rejected
	legacyComponent := gossr.Render(
		`<ul>@for index, value range Items {<li>${index}:${value}</li>}</ul>`,
		Props{Items: []string{"x"}},
	)
	var sb strings.Builder
	err := legacyComponent.Render(&sb)
	if err == nil || !strings.Contains(err.Error(), "malformed @for header") {
		testRunner.Fatalf("Expected legacy 'range' keyword to return malformed @for header error, got %v", err)
	}
}

func TestControlFlowSwitchAndTypeSwitch(testRunner *testing.T) {
	type Props struct {
		State   string
		Payload any
	}

	component := gossr.Render(
		`@switch State { @case "draft":<span>Draft</span> @case "ready", "pending":<span>Active</span> @default:<span>Other</span> }`,
		Props{State: "ready"},
	)
	if !strings.Contains(component.String(), `<span>Active</span>`) {
		testRunner.Fatalf("Expected @switch value branch to render, got %q", component.String())
	}

	booleanComponent := gossr.Render(
		`@switch { @case len(State) > 0:<span>Has Value</span> @default:<span>Empty</span> }`,
		Props{State: "x"},
	)
	if !strings.Contains(booleanComponent.String(), `<span>Has Value</span>`) {
		testRunner.Fatalf("Expected truthy @switch branch to render, got %q", booleanComponent.String())
	}

	typeSwitchComponent := gossr.Render(
		`@switch Payload := Payload.(type) { case int:<p>int</p> case string:<p>string</p> default:<p>other</p> }`,
		Props{Payload: 42},
	)
	if !strings.Contains(typeSwitchComponent.String(), `<p>int</p>`) {
		testRunner.Fatalf("Expected type switch branch to render, got %q", typeSwitchComponent.String())
	}
}

func TestControlFlowDefer(testRunner *testing.T) {
	type Props struct {
		Error any
	}

	// Scenario 1: Error present -> @if error triggers, renders error message, @return returns execution and skips post-if text
	withErrorComponent := gossr.Render(
		`<h1>Header</h1>@defer error { @if error { <p>something is wrong, ${error}</p> @return } <span>execution here if not returned</span> }`,
		Props{Error: "invalid input"},
	)
	outputWithError := withErrorComponent.String()
	if !strings.Contains(outputWithError, `<h1>Header</h1>`) || !strings.Contains(outputWithError, `<p>something is wrong, invalid input</p>`) {
		testRunner.Fatalf("Expected @defer error branch to render header and error message, got %q", outputWithError)
	}
	if strings.Contains(outputWithError, `execution here if not returned`) {
		testRunner.Fatalf("Expected @return inside @defer to stop execution of defer body, got %q", outputWithError)
	}

	// Scenario 2: Error present, testing ${err} alias
	aliasErrorComponent := gossr.Render(
		`@defer error { @if error { <p>error is ${err}</p> @return } }`,
		Props{Error: "database failed"},
	)
	if !strings.Contains(aliasErrorComponent.String(), `<p>error is database failed</p>`) {
		testRunner.Fatalf("Expected ${err} alias to work in @defer, got %q", aliasErrorComponent.String())
	}

	// Scenario 3: No error present -> @if error is skipped, post-if text inside @defer renders
	noErrorComponent := gossr.Render(
		`<h1>Header</h1>@defer error { @if error { <p>something is wrong, ${error}</p> @return } <span>execution here if not returned</span> }`,
		Props{Error: nil},
	)
	outputNoError := noErrorComponent.String()
	if !strings.Contains(outputNoError, `<h1>Header</h1>`) || !strings.Contains(outputNoError, `<span>execution here if not returned</span>`) {
		testRunner.Fatalf("Expected fallback execution inside @defer when error is nil, got %q", outputNoError)
	}
	if strings.Contains(outputNoError, `something is wrong`) {
		testRunner.Fatalf("Expected @if error to be skipped when error is nil, got %q", outputNoError)
	}

	// Scenario 4: LIFO execution order of multiple defers
	lifoComponent := gossr.Render(
		`start|@defer {first|}@defer {second|}end|`,
		nil,
	)
	outputLifo := lifoComponent.String()
	expectedLifo := `start|end|second|first|`
	if outputLifo != expectedLifo {
		testRunner.Fatalf("Expected LIFO deferred order %q, got %q", expectedLifo, outputLifo)
	}
}

func TestControlFlowPanic(testRunner *testing.T) {
	type Props struct {
		Reason string
	}

	// Scenario 1: Unhandled @panic -> error is returned from component
	unhandledComp := gossr.Render(
		`<div>Start</div>@panic "fatal system error"<div>End</div>`,
		nil,
	)
	var sb strings.Builder
	err := unhandledComp.Render(&sb)
	if err == nil || !strings.Contains(err.Error(), "fatal system error") {
		testRunner.Fatalf("Expected unhandled @panic to return error, got %v", err)
	}

	// Scenario 2: Handled @panic via @defer -> panic is caught, @defer renders error HTML
	handledComp := gossr.Render(
		`@defer error { @if error { <div class="error">Recovered: ${error}</div> @return } }<h1>App</h1>@panic Reason`,
		Props{Reason: "corrupted state"},
	)
	outputHandled := handledComp.String()
	if !strings.Contains(outputHandled, `<h1>App</h1>`) || !strings.Contains(outputHandled, `<div class="error">Recovered: corrupted state</div>`) {
		testRunner.Fatalf("Expected @defer to recover from @panic and render error HTML, got %q", outputHandled)
	}
}


