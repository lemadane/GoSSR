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
		`<ul>@for index, value range Items {@if value == "skip" { @continue }<li>${index}:${value}</li>@if value == "stop" { @break }}</ul>`,
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
		`<ul>@for index, value range Items {<li>${index}:${value}</li>} @else {<li>empty</li>}</ul>`,
		Props{Items: nil},
	)

	if !strings.Contains(emptyComponent.String(), `<li>empty</li>`) {
		testRunner.Fatalf("Expected @for @else fallback for empty collection, got %q", emptyComponent.String())
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
