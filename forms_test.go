package gossr_test

import (
	"strings"
	"testing"

	"github.com/lemadane/gossr"
)

func TestFormInputAttributeQuoteProtection(testRunner *testing.T) {
	type UserFormProps struct {
		Name  string
		Email string
	}

	comp := gossr.Render(
		`<form>`+
			`<input type="text" name="name" value="${properties.Name}" />`+
			`<input type="email" name="email" value="${properties.Email}" />`+
			`</form>`,
		UserFormProps{
			Name:  `Sarah "The Boss" Connor`,
			Email: `sarah<script>@example.com`,
		},
	)

	output := comp.String()

	if !strings.Contains(output, `value="Sarah &quot;The Boss&quot; Connor"`) && !strings.Contains(output, `value="Sarah &#34;The Boss&#34; Connor"`) {
		testRunner.Errorf("Expected input value attribute to escape quotes safely, got %q", output)
	}

	expectedEmail := `value="sarah&lt;script&gt;@example.com"`
	if !strings.Contains(output, expectedEmail) {
		testRunner.Errorf("Expected input email attribute to escape tags safely %q, got %q", expectedEmail, output)
	}
}

func TestCheckboxAndRadioButtonRendering(testRunner *testing.T) {
	type PreferencesProps struct {
		Subscribe bool
		Role      string
	}

	comp := gossr.Render(
		`<form>`+
			`<input type="checkbox" name="subscribe" ${properties.Subscribe ? "checked" : ""} />`+
			`<input type="radio" name="role" value="ADMIN" ${properties.Role == "ADMIN" ? "checked" : ""} />`+
			`<input type="radio" name="role" value="USER" ${properties.Role == "USER" ? "checked" : ""} />`+
			`</form>`,
		PreferencesProps{
			Subscribe: true,
			Role:      "ADMIN",
		},
	)

	output := comp.String()

	if !strings.Contains(output, `<input type="checkbox" name="subscribe" checked />`) {
		testRunner.Errorf("Expected checkbox to render 'checked', got %q", output)
	}

	if !strings.Contains(output, `<input type="radio" name="role" value="ADMIN" checked />`) {
		testRunner.Errorf("Expected ADMIN radio button to render 'checked', got %q", output)
	}

	if strings.Contains(output, `<input type="radio" name="role" value="USER" checked`) {
		testRunner.Errorf("Expected USER radio button NOT to render 'checked', got %q", output)
	}
}

func TestFormSubmissionValidationErrorFeedback(testRunner *testing.T) {
	type FormFeedbackProps struct {
		Name         string
		Email        string
		ErrorMessage string
	}

	comp := gossr.Render(
		`<div id="form-container">`+
			`<p class="error">${properties.ErrorMessage}</p>`+
			`<form hx-post="/users">`+
			`<input name="name" value="${properties.Name}" />`+
			`<input name="email" value="${properties.Email}" />`+
			`<button type="submit">Submit</button>`+
			`</form>`+
			`</div>`,
		FormFeedbackProps{
			Name:         "John Doe",
			Email:        "john@invalid",
			ErrorMessage: "Email format is invalid & missing domain",
		},
	)

	output := comp.String()

	expectedError := `Email format is invalid &amp; missing domain`
	if !strings.Contains(output, expectedError) {
		testRunner.Errorf("Expected form error message to be escaped %q, got %q", expectedError, output)
	}

	if !strings.Contains(output, `value="John Doe"`) {
		testRunner.Errorf("Expected form input to preserve pre-filled value, got %q", output)
	}
}
