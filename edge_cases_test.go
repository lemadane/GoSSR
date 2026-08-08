package gossr_test

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/lemadane/gossr"
)

func TestRawHtmlOfAndSafeUrlOfAndRegisterTag(testRunner *testing.T) {
	gossr.SetStrict(false)
	gossr.RegisterTag("TestTagHelper", func(props map[string]string) gossr.SSR {
		return gossr.Render(`<span>TagHelper</span>`)
	})

	raw := gossr.RawHtmlOf("<b>raw html of</b>")
	safe := gossr.SafeUrlOf("https://example.com/safe")

	comp := gossr.Render(`<div>${properties.Raw} | ${properties.Safe} | <TestTagHelper /></div>`, struct {
		Raw  gossr.RawHtml
		Safe gossr.SafeUrl
	}{
		Raw:  raw,
		Safe: safe,
	})

	output := comp.String()
	if !strings.Contains(output, "<b>raw html of</b>") {
		testRunner.Errorf("Expected RawHtmlOf output, got %q", output)
	}
	if !strings.Contains(output, "https://example.com/safe") {
		testRunner.Errorf("Expected SafeUrlOf output, got %q", output)
	}
	if !strings.Contains(output, "<span>TagHelper</span>") {
		testRunner.Errorf("Expected RegisterTag output, got %q", output)
	}
}

type errWriter struct{}

func (w errWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("simulated write error")
}

func TestDirectWriterRenderingRawHtmlAndSafeUrl(testRunner *testing.T) {
	raw := gossr.Raw("<b>bold</b>")
	safe := gossr.URL("javascript:alert(1)")

	var sb strings.Builder
	if err := raw.Render(&sb); err != nil {
		testRunner.Fatalf("Unexpected RawHtml.Render error: %v", err)
	}
	if sb.String() != "<b>bold</b>" {
		testRunner.Errorf("Expected '<b>bold</b>', got %q", sb.String())
	}

	sb.Reset()
	if err := safe.Render(&sb); err != nil {
		testRunner.Fatalf("Unexpected SafeUrl.Render error: %v", err)
	}
	if sb.String() != "about:blank" {
		testRunner.Errorf("Expected 'about:blank', got %q", sb.String())
	}

	failWriter := errWriter{}
	if err := raw.Render(failWriter); err == nil {
		testRunner.Error("Expected error when rendering RawHtml to failing writer, got nil")
	}
	if err := safe.Render(failWriter); err == nil {
		testRunner.Error("Expected error when rendering SafeUrl to failing writer, got nil")
	}
}

func TestWriterFailuresInComponentRender(testRunner *testing.T) {
	comp := gossr.Render(`<div>Hello World</div>`)
	failWriter := errWriter{}

	err := comp.Render(failWriter)
	if err == nil {
		testRunner.Error("Expected component.Render to return error when writing to failing writer, got nil")
	}
}

func TestTruthinessEvaluationAllTypes(testRunner *testing.T) {
	type Scope struct {
		IntZero    int
		IntVal     int
		UintZero   uint
		UintVal    uint
		FloatZero  float64
		FloatVal   float64
		BoolFalse  bool
		BoolTrue   bool
		StrEmpty   string
		StrVal     string
		SliceEmpty []string
		SliceVal   []string
		MapEmpty   map[string]int
		MapVal     map[string]int
		NilPtr     *string
		NonNilPtr  *string
	}

	val := "hello"
	scope := Scope{
		IntZero:    0,
		IntVal:     42,
		UintZero:   0,
		UintVal:    100,
		FloatZero:  0.0,
		FloatVal:   3.14,
		BoolFalse:  false,
		BoolTrue:   true,
		StrEmpty:   "",
		StrVal:     "content",
		SliceEmpty: []string{},
		SliceVal:   []string{"a"},
		MapEmpty:   map[string]int{},
		MapVal:     map[string]int{"a": 1},
		NilPtr:     nil,
		NonNilPtr:  &val,
	}

	tpl := `<IntZero>${properties.IntZero ? "Y" : "N"}</IntZero>` +
		`<IntVal>${properties.IntVal ? "Y" : "N"}</IntVal>` +
		`<UintZero>${properties.UintZero ? "Y" : "N"}</UintZero>` +
		`<UintVal>${properties.UintVal ? "Y" : "N"}</UintVal>` +
		`<FloatZero>${properties.FloatZero ? "Y" : "N"}</FloatZero>` +
		`<FloatVal>${properties.FloatVal ? "Y" : "N"}</FloatVal>` +
		`<BoolFalse>${properties.BoolFalse ? "Y" : "N"}</BoolFalse>` +
		`<BoolTrue>${properties.BoolTrue ? "Y" : "N"}</BoolTrue>` +
		`<StrEmpty>${properties.StrEmpty ? "Y" : "N"}</StrEmpty>` +
		`<StrVal>${properties.StrVal ? "Y" : "N"}</StrVal>` +
		`<SliceEmpty>${properties.SliceEmpty ? "Y" : "N"}</SliceEmpty>` +
		`<SliceVal>${properties.SliceVal ? "Y" : "N"}</SliceVal>` +
		`<MapEmpty>${properties.MapEmpty ? "Y" : "N"}</MapEmpty>` +
		`<MapVal>${properties.MapVal ? "Y" : "N"}</MapVal>` +
		`<NilPtr>${properties.NilPtr ? "Y" : "N"}</NilPtr>` +
		`<NonNilPtr>${properties.NonNilPtr ? "Y" : "N"}</NonNilPtr>`

	comp := gossr.Render(tpl, scope)
	output := comp.String()

	expected := `<IntZero>N</IntZero>` +
		`<IntVal>Y</IntVal>` +
		`<UintZero>N</UintZero>` +
		`<UintVal>Y</UintVal>` +
		`<FloatZero>N</FloatZero>` +
		`<FloatVal>Y</FloatVal>` +
		`<BoolFalse>N</BoolFalse>` +
		`<BoolTrue>Y</BoolTrue>` +
		`<StrEmpty>N</StrEmpty>` +
		`<StrVal>Y</StrVal>` +
		`<SliceEmpty>N</SliceEmpty>` +
		`<SliceVal>Y</SliceVal>` +
		`<MapEmpty>N</MapEmpty>` +
		`<MapVal>Y</MapVal>` +
		`<NilPtr>N</NilPtr>` +
		`<NonNilPtr>Y</NonNilPtr>`

	if output != expected {
		testRunner.Errorf("Truthiness evaluation output mismatch.\nExpected: %s\nGot:      %s", expected, output)
	}
}

func TestTernaryInequalityAndUnquoted(testRunner *testing.T) {
	type Scope struct {
		Role   string
		Status string
	}

	scope := Scope{Role: "DEVELOPER", Status: "ACTIVE"}

	comp := gossr.Render(
		`<div>${properties.Role != "ADMIN" ? "UserAccess" : "AdminAccess"} | ${properties.Status == "ACTIVE" ? 1 : 0}</div>`,
		scope,
	)

	output := comp.String()
	if !strings.Contains(output, "UserAccess | 1") {
		testRunner.Errorf("Expected 'UserAccess | 1', got %q", output)
	}
}

func TestMapAndPointerPropertyResolutionEdgeCases(testRunner *testing.T) {
	type User struct {
		Name string
	}
	type Props struct {
		MapData    map[string]any
		UserPtr    *User
		UserPtrPtr **User
		ArrayData  [2]string
	}

	u := User{Name: "PointerUser"}
	uPtr := &u
	uPtrPtr := &uPtr

	scope := Props{
		MapData: map[string]any{
			"config": map[string]string{
				"theme": "dark",
			},
		},
		UserPtr:    uPtr,
		UserPtrPtr: uPtrPtr,
		ArrayData:  [2]string{"Alpha", "Beta"},
	}

	comp := gossr.Render(
		`<div>Theme: ${properties.MapData.config.theme} | User: ${properties.UserPtr.Name} | DeepUser: ${properties.UserPtrPtr.Name} | Array: ${properties.ArrayData.map(a => <span>${a}</span>)}</div>`,
		scope,
	)

	output := comp.String()
	expected := `<div>Theme: dark | User: PointerUser | DeepUser: PointerUser | Array: <span>Alpha</span><span>Beta</span></div>`

	if output != expected {
		testRunner.Errorf("Expected edge case property resolution %q, got %q", expected, output)
	}
}

func TestUnicodeAndSpecialCharsInPropertyNamesAndValues(testRunner *testing.T) {
	type SpecialProps struct {
		User_Name_1 string
		Japanese    string
		Emoji       string
		XmlQuotes   string
	}

	scope := SpecialProps{
		User_Name_1: "user_123",
		Japanese:    "こんにちは世界",
		Emoji:       "🚀 GoSSR",
		XmlQuotes:   `"Hello" & <World>`,
	}

	comp := gossr.Render(
		`<div>${properties.User_Name_1} | ${properties.Japanese} | ${properties.Emoji} | ${properties.XmlQuotes}</div>`,
		scope,
	)

	output := comp.String()
	if !strings.Contains(output, "こんにちは世界") {
		testRunner.Errorf("Expected Japanese text preserved, got %q", output)
	}
	if !strings.Contains(output, "🚀 GoSSR") {
		testRunner.Errorf("Expected Emoji preserved, got %q", output)
	}
	if !strings.Contains(output, `&quot;Hello&quot; &amp; &lt;World&gt;`) && !strings.Contains(output, `&#34;Hello&#34; &amp; &lt;World&gt;`) {
		testRunner.Errorf("Expected XML special chars escaped, got %q", output)
	}
}

func TestVeryLargeSliceMapping(testRunner *testing.T) {
	items := make([]string, 10000)
	for i := 0; i < 10000; i++ {
		items[i] = fmt.Sprintf("item-%d", i)
	}

	type Props struct {
		List []string
	}

	comp := gossr.Render(`<div>${properties.List.map(i => <span>${i}</span>)}</div>`, Props{List: items})
	output := comp.String()

	if !strings.Contains(output, "<span>item-0</span>") || !strings.Contains(output, "<span>item-9999</span>") {
		testRunner.Errorf("Expected 10,000 item slice mapping to contain first and last items, length of output: %d", len(output))
	}
}

func TestConcurrentRenderingThreadSafety(testRunner *testing.T) {
	type Props struct {
		Title string
		Count int
	}

	gossr.Register("ConcurrentTag", func(p Props) gossr.SSR {
		return gossr.Render(`<div class="concurrent">${properties.Title}: ${properties.Count}</div>`, p)
	})

	const numGoroutines = 100
	const iterationsPerGoroutine = 50

	var waitGroup sync.WaitGroup
	waitGroup.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(routineIndex int) {
			defer waitGroup.Done()
			for i := 0; i < iterationsPerGoroutine; i++ {
				props := Props{Title: "Worker", Count: routineIndex*1000 + i}
				comp := gossr.Render(`<div><ConcurrentTag title="${properties.Title}" count="${properties.Count}" /></div>`, props)
				output := comp.String()

				expectedSub := fmt.Sprintf("Worker: %d", routineIndex*1000+i)
				if !strings.Contains(output, expectedSub) {
					testRunner.Errorf("Concurrent rendering mismatch for routine %d iteration %d: got %q", routineIndex, i, output)
					return
				}
			}
		}(g)
	}

	waitGroup.Wait()
}

func TestFormatAndEscapeValueDirectCall(testRunner *testing.T) {
	type SliceProps struct {
		Items []string
	}
	comp := gossr.Render(`<span>${properties.Items}</span>`, SliceProps{Items: []string{"A", "B"}})
	if comp.String() != `<span>AB</span>` {
		testRunner.Errorf("Expected '<span>AB</span>', got %q", comp.String())
	}
}
