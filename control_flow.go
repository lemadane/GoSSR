package gossr

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type controlFlowSignal int

const (
	controlFlowSignalNone controlFlowSignal = iota
	controlFlowSignalBreak
	controlFlowSignalContinue
	controlFlowSignalReturn
)

func (signal controlFlowSignal) String() string {
	switch signal {
	case controlFlowSignalBreak:
		return "break"
	case controlFlowSignalContinue:
		return "continue"
	case controlFlowSignalReturn:
		return "return"
	default:
		return "none"
	}
}

func containsControlFlowDirective(templateString string) bool {
	return strings.Contains(templateString, "@if") || strings.Contains(templateString, "@for") || strings.Contains(templateString, "@switch") ||
		strings.Contains(templateString, "@defer") || strings.Contains(templateString, "@panic") || strings.Contains(templateString, "@break") || strings.Contains(templateString, "@continue") || strings.Contains(templateString, "@return")
}

func renderTemplateWithControlFlow(templateString string, scopeArguments []any, depth int) (string, error) {
	output, signal, err := renderControlFlowFragment(templateString, scopeArguments, nil, depth)
	if err != nil {
		return "", err
	}
	if signal == controlFlowSignalBreak || signal == controlFlowSignalContinue {
		return "", fmt.Errorf("GoSSR control flow directive @%s is only valid inside @for blocks", signal.String())
	}
	return output, nil
}

type deferredBlock struct {
	header string
	body   string
}

func renderDeferDirective(templateString string, startIndex int) (string, string, int, error) {
	position := skipWhitespace(templateString, startIndex+len("@defer"))
	headEnd := strings.IndexByte(templateString[position:], '{')
	if headEnd == -1 {
		return "", "", 0, fmt.Errorf("GoSSR malformed @defer directive")
	}
	header := strings.TrimSpace(templateString[position : position+headEnd])
	body, nextIndex, err := consumeBalancedBlock(templateString, position+headEnd)
	if err != nil {
		return "", "", 0, err
	}
	return header, body, nextIndex, nil
}

func renderPanicDirective(templateString string, startIndex int, scopeArguments []any, locals map[string]any) (error, int, error) {
	position := skipWhitespace(templateString, startIndex+len("@panic"))
	endIndex := position
	for endIndex < len(templateString) {
		ch := templateString[endIndex]
		if ch == '\n' || ch == '\r' || ch == ';' || ch == '}' || ch == '<' || ch == '@' {
			break
		}
		endIndex++
	}
	expr := strings.TrimSpace(templateString[position:endIndex])

	var panicMessage any = "GoSSR template panic"
	if expr != "" {
		val, found, err := evaluateControlExpression(expr, scopeArguments, locals, true)
		if err == nil && found && val != nil {
			panicMessage = val
		} else {
			panicMessage = expr
		}
	}

	return fmt.Errorf("%v", panicMessage), endIndex, nil
}

func executeDeferredBlocks(defers []deferredBlock, scopeArguments []any, locals map[string]any, depth int, renderErr error) (string, error) {
	if len(defers) == 0 {
		return "", renderErr
	}

	var deferSb strings.Builder
	for i := len(defers) - 1; i >= 0; i-- {
		d := defers[i]
		deferLocals := cloneLocals(locals)
		if renderErr != nil {
			deferLocals["error"] = renderErr.Error()
			deferLocals["err"] = renderErr.Error()
		}

		if d.header != "" {
			val, found, _ := evaluateControlExpression(d.header, scopeArguments, deferLocals, false)
			if found && val != nil {
				if _, hasErr := deferLocals["error"]; !hasErr {
					deferLocals["error"] = val
					deferLocals["err"] = val
				}
			}
		}

		deferHtml, _, err := renderControlFlowFragment(d.body, scopeArguments, deferLocals, depth)
		if err != nil {
			return deferSb.String(), err
		}
		deferSb.WriteString(deferHtml)
	}

	return deferSb.String(), nil
}

func renderControlFlowFragment(templateString string, scopeArguments []any, locals map[string]any, depth int) (string, controlFlowSignal, error) {
	if templateString == "" {
		return "", controlFlowSignalNone, nil
	}

	var sb strings.Builder
	var defers []deferredBlock
	position := 0

	var signal controlFlowSignal
	var renderErr error

	for position < len(templateString) {
		directiveIndex := findNextTopLevelDirective(templateString, position)
		if directiveIndex == -1 {
			fragment, err := renderTemplateFragment(templateString[position:], scopeArguments, locals, depth)
			if err != nil {
				renderErr = err
				break
			}
			sb.WriteString(fragment)
			break
		}

		if directiveIndex > position {
			fragment, err := renderTemplateFragment(templateString[position:directiveIndex], scopeArguments, locals, depth)
			if err != nil {
				renderErr = err
				break
			}
			sb.WriteString(fragment)
		}

		switch {
		case matchesDirectiveAt(templateString, directiveIndex, "if"):
			branchHtml, nextIndex, sig, err := renderIfDirective(templateString, directiveIndex, scopeArguments, locals, depth)
			if err != nil {
				renderErr = err
				goto done
			}
			sb.WriteString(branchHtml)
			position = nextIndex
			if sig != controlFlowSignalNone {
				signal = sig
				goto done
			}
			continue
		case matchesDirectiveAt(templateString, directiveIndex, "for"):
			loopHtml, nextIndex, sig, err := renderForDirective(templateString, directiveIndex, scopeArguments, locals, depth)
			if err != nil {
				renderErr = err
				goto done
			}
			sb.WriteString(loopHtml)
			position = nextIndex
			if sig != controlFlowSignalNone {
				signal = sig
				goto done
			}
			continue
		case matchesDirectiveAt(templateString, directiveIndex, "switch"):
			switchHtml, nextIndex, sig, err := renderSwitchDirective(templateString, directiveIndex, scopeArguments, locals, depth)
			if err != nil {
				renderErr = err
				goto done
			}
			sb.WriteString(switchHtml)
			position = nextIndex
			if sig != controlFlowSignalNone {
				signal = sig
				goto done
			}
			continue
		case matchesDirectiveAt(templateString, directiveIndex, "defer"):
			header, body, nextIndex, err := renderDeferDirective(templateString, directiveIndex)
			if err != nil {
				renderErr = err
				goto done
			}
			defers = append(defers, deferredBlock{header: header, body: body})
			position = nextIndex
			continue
		case matchesDirectiveAt(templateString, directiveIndex, "panic"):
			panicErr, nextIndex, err := renderPanicDirective(templateString, directiveIndex, scopeArguments, locals)
			if err != nil {
				renderErr = err
				goto done
			}
			renderErr = panicErr
			position = nextIndex
			goto done
		case matchesDirectiveAt(templateString, directiveIndex, "break"):
			signal = controlFlowSignalBreak
			goto done
		case matchesDirectiveAt(templateString, directiveIndex, "continue"):
			signal = controlFlowSignalContinue
			goto done
		case matchesDirectiveAt(templateString, directiveIndex, "return"):
			signal = controlFlowSignalReturn
			goto done
		default:
			fragment, err := renderTemplateFragment(templateString[directiveIndex:directiveIndex+1], scopeArguments, locals, depth)
			if err != nil {
				renderErr = err
				goto done
			}
			sb.WriteString(fragment)
			position = directiveIndex + 1
		}
	}

done:
	if len(defers) > 0 {
		deferredOutput, err := executeDeferredBlocks(defers, scopeArguments, locals, depth, renderErr)
		if err != nil {
			return "", controlFlowSignalNone, err
		}
		if renderErr != nil {
			if deferredOutput != "" {
				sb.WriteString(deferredOutput)
				return sb.String(), controlFlowSignalNone, nil
			}
			return "", controlFlowSignalNone, renderErr
		}
		sb.WriteString(deferredOutput)
	} else if renderErr != nil {
		return "", controlFlowSignalNone, renderErr
	}

	return sb.String(), signal, nil
}

func renderTemplateFragment(templateString string, scopeArguments []any, locals map[string]any, depth int) (string, error) {
	if templateString == "" {
		return "", nil
	}

	combinedScopeArguments := scopeArguments
	if len(locals) > 0 {
		combinedScopeArguments = append([]any{locals}, scopeArguments...)
	}

	outputHtml := templateString
	for _, argument := range combinedScopeArguments {
		var err error
		outputHtml, err = processScopeArgumentProperties(outputHtml, argument)
		if err != nil {
			return "", err
		}
	}

	outputHtml, err := processNamedScopeArgumentProperties(outputHtml, combinedScopeArguments)
	if err != nil {
		return "", err
	}

	processedHtml, err := processCustomComponentTags(outputHtml, depth)
	if err != nil {
		return "", err
	}

	return processedHtml, nil
}

func processNamedScopeArgumentProperties(templateString string, scopeArguments []any) (string, error) {
	itemMatches := genericItemPropertyRegex.FindAllStringSubmatchIndex(templateString, -1)
	if len(itemMatches) == 0 {
		return templateString, nil
	}

	var sb strings.Builder
	lastIndex := 0
	for _, loc := range itemMatches {
		matchStart, matchEnd := loc[0], loc[1]
		prefix := templateString[loc[2]:loc[3]]
		if prefix == "properties" {
			continue
		}

		sb.WriteString(templateString[lastIndex:matchStart])

		path := []string{}
		pathString := ""
		if len(loc) >= 6 && loc[4] != -1 && loc[5] != -1 {
			pathString = templateString[loc[4]:loc[5]]
			if pathString != "" {
				path = strings.Split(pathString, ".")
			}
		}

		value, found := resolveNamedScopePath(scopeArguments, prefix, path)
		if found {
			attrContext := getInterpolationAttributeContext(templateString, matchStart)
			if err := formatAndRenderValueForContext(&sb, value, attrContext); err != nil {
				return "", err
			}
		} else {
			if isStrict() {
				if pathString != "" {
					return "", fmt.Errorf("GoSSR strict mode error: unresolved property path '${%s.%s}'", prefix, pathString)
				}
				return "", fmt.Errorf("GoSSR strict mode error: unresolved property path '${%s}'", prefix)
			}
			sb.WriteString(templateString[matchStart:matchEnd])
		}
		lastIndex = matchEnd
	}
	if lastIndex < len(templateString) {
		sb.WriteString(templateString[lastIndex:])
	}

	return sb.String(), nil
}

func resolveNamedScopePath(scopeArguments []any, prefix string, path []string) (any, bool) {
	searchPath := append([]string{prefix}, path...)
	for _, argument := range scopeArguments {
		if argument == nil {
			continue
		}
		if value, found := resolvePropertyPath(argument, searchPath); found {
			return value, true
		}
	}
	return nil, false
}

func findNextTopLevelDirective(templateString string, startIndex int) int {
	for index := startIndex; index < len(templateString); index++ {
		if templateString[index] != '@' {
			continue
		}
		if matchesDirectiveAt(templateString, index, "if") || matchesDirectiveAt(templateString, index, "for") || matchesDirectiveAt(templateString, index, "switch") ||
			matchesDirectiveAt(templateString, index, "defer") || matchesDirectiveAt(templateString, index, "panic") || matchesDirectiveAt(templateString, index, "break") || matchesDirectiveAt(templateString, index, "continue") || matchesDirectiveAt(templateString, index, "return") {
			return index
		}
	}
	return -1
}

func matchesDirectiveAt(templateString string, index int, keyword string) bool {
	if index < 0 || index >= len(templateString) || templateString[index] != '@' {
		return false
	}
	if !strings.HasPrefix(templateString[index+1:], keyword) {
		return false
	}
	endIndex := index + 1 + len(keyword)
	if endIndex < len(templateString) && isDirectiveNameCharacter(templateString[endIndex]) {
		return false
	}
	if index > 0 {
		previous := templateString[index-1]
		if isDirectiveNameCharacter(previous) {
			return false
		}
	}
	return true
}

func isDirectiveNameCharacter(charByte byte) bool {
	return (charByte >= 'a' && charByte <= 'z') || (charByte >= 'A' && charByte <= 'Z') || (charByte >= '0' && charByte <= '9') || charByte == '_'
}

func skipWhitespace(templateString string, startIndex int) int {
	for startIndex < len(templateString) {
		current := templateString[startIndex]
		if current != ' ' && current != '\t' && current != '\n' && current != '\r' {
			break
		}
		startIndex++
	}
	return startIndex
}

func consumeBalancedBlock(templateString string, openBraceIndex int) (string, int, error) {
	if openBraceIndex < 0 || openBraceIndex >= len(templateString) || templateString[openBraceIndex] != '{' {
		return "", 0, fmt.Errorf("GoSSR expected '{' at block start")
	}

	depth := 1
	quoteChar := byte(0)
	for index := openBraceIndex + 1; index < len(templateString); index++ {
		current := templateString[index]
		if quoteChar != 0 {
			if current == quoteChar {
				quoteChar = 0
			}
			continue
		}

		switch current {
		case '"', '\'', '`':
			quoteChar = current
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return templateString[openBraceIndex+1 : index], index + 1, nil
			}
		}
	}

	return "", 0, fmt.Errorf("GoSSR unclosed block starting at index %d", openBraceIndex)
}

func renderIfDirective(templateString string, startIndex int, scopeArguments []any, locals map[string]any, depth int) (string, int, controlFlowSignal, error) {
	position := skipWhitespace(templateString, startIndex+len("@if"))
	headEnd := strings.IndexByte(templateString[position:], '{')
	if headEnd == -1 {
		return "", 0, controlFlowSignalNone, fmt.Errorf("GoSSR malformed @if directive")
	}
	conditionExpression := strings.TrimSpace(templateString[position : position+headEnd])
	body, nextIndex, err := consumeBalancedBlock(templateString, position+headEnd)
	if err != nil {
		return "", 0, controlFlowSignalNone, err
	}

	matched := false
	branchOutput := ""
	branchSignal := controlFlowSignalNone

	conditionValue, found, err := evaluateControlExpression(conditionExpression, scopeArguments, locals, false)
	if err != nil {
		return "", 0, controlFlowSignalNone, err
	}
	if found && evaluateTruthiness(conditionValue) {
		matched = true
		branchOutput, branchSignal, err = renderControlFlowFragment(body, scopeArguments, locals, depth)
		if err != nil {
			return "", 0, controlFlowSignalNone, err
		}
	}

	position = skipWhitespace(templateString, nextIndex)
	for !matched && matchesDirectiveAt(templateString, position, "elseif") {
		position = skipWhitespace(templateString, position+len("@elseif"))
		headEnd = strings.IndexByte(templateString[position:], '{')
		if headEnd == -1 {
			return "", 0, controlFlowSignalNone, fmt.Errorf("GoSSR malformed @elseif directive")
		}
		conditionExpression = strings.TrimSpace(templateString[position : position+headEnd])
		body, nextIndex, err = consumeBalancedBlock(templateString, position+headEnd)
		if err != nil {
			return "", 0, controlFlowSignalNone, err
		}

		conditionValue, found, err = evaluateControlExpression(conditionExpression, scopeArguments, locals, false)
		if err != nil {
			return "", 0, controlFlowSignalNone, err
		}
		if found && evaluateTruthiness(conditionValue) {
			matched = true
			branchOutput, branchSignal, err = renderControlFlowFragment(body, scopeArguments, locals, depth)
			if err != nil {
				return "", 0, controlFlowSignalNone, err
			}
		}
		position = skipWhitespace(templateString, nextIndex)
	}

	if !matched && matchesDirectiveAt(templateString, position, "else") {
		position = skipWhitespace(templateString, position+len("@else"))
		if position >= len(templateString) || templateString[position] != '{' {
			return "", 0, controlFlowSignalNone, fmt.Errorf("GoSSR malformed @else directive")
		}
		body, nextIndex, err = consumeBalancedBlock(templateString, position)
		if err != nil {
			return "", 0, controlFlowSignalNone, err
		}
		branchOutput, branchSignal, err = renderControlFlowFragment(body, scopeArguments, locals, depth)
		if err != nil {
			return "", 0, controlFlowSignalNone, err
		}
		position = nextIndex
	}

	return branchOutput, position, branchSignal, nil
}

func renderForDirective(templateString string, startIndex int, scopeArguments []any, locals map[string]any, depth int) (string, int, controlFlowSignal, error) {
	position := skipWhitespace(templateString, startIndex+len("@for"))
	headEnd := strings.IndexByte(templateString[position:], '{')
	if headEnd == -1 {
		return "", 0, controlFlowSignalNone, fmt.Errorf("GoSSR malformed @for directive")
	}
	header := strings.TrimSpace(templateString[position : position+headEnd])
	body, nextIndex, err := consumeBalancedBlock(templateString, position+headEnd)
	if err != nil {
		return "", 0, controlFlowSignalNone, err
	}

	loopIndexName, loopValueName, rangeExpression, err := parseForHeader(header)
	if err != nil {
		return "", 0, controlFlowSignalNone, err
	}

	iterableValue, found, err := evaluateControlExpression(rangeExpression, scopeArguments, locals, false)
	if err != nil {
		return "", 0, controlFlowSignalNone, err
	}

	output := strings.Builder{}
	hasItems := false
	if found {
		collection := reflect.ValueOf(iterableValue)
		for collection.Kind() == reflect.Pointer || collection.Kind() == reflect.Interface {
			if collection.IsNil() {
				collection = reflect.Value{}
				break
			}
			collection = collection.Elem()
		}

		if collection.IsValid() {
			switch collection.Kind() {
			case reflect.Slice, reflect.Array:
				hasItems = collection.Len() > 0
				for itemIndex := 0; itemIndex < collection.Len(); itemIndex++ {
					itemValue := collection.Index(itemIndex).Interface()
					childLocals := cloneLocals(locals)
					childLocals[loopIndexName] = itemIndex
					childLocals[loopValueName] = itemValue
					childLocals["index"] = itemIndex
					childLocals["value"] = itemValue
					childLocals["item"] = itemValue

					fragment, signal, err := renderControlFlowFragment(body, scopeArguments, childLocals, depth)
					if err != nil {
						return "", 0, controlFlowSignalNone, err
					}
					output.WriteString(fragment)
					switch signal {
					case controlFlowSignalNone:
					case controlFlowSignalContinue:
						continue
					case controlFlowSignalBreak:
						position = nextIndex
						goto loopDone
					case controlFlowSignalReturn:
						position = nextIndex
						return output.String(), position, controlFlowSignalReturn, nil
					}
				}
			case reflect.Map:
				hasItems = collection.Len() > 0
				keys := collection.MapKeys()
				for itemIndex, keyValue := range keys {
					itemValue := collection.MapIndex(keyValue).Interface()
					childLocals := cloneLocals(locals)
					childLocals[loopIndexName] = itemIndex
					childLocals[loopValueName] = itemValue
					childLocals["index"] = itemIndex
					childLocals["value"] = itemValue
					childLocals["item"] = itemValue
					childLocals["key"] = keyValue.Interface()

					fragment, signal, err := renderControlFlowFragment(body, scopeArguments, childLocals, depth)
					if err != nil {
						return "", 0, controlFlowSignalNone, err
					}
					output.WriteString(fragment)
					switch signal {
					case controlFlowSignalNone:
					case controlFlowSignalContinue:
						continue
					case controlFlowSignalBreak:
						position = nextIndex
						goto loopDone
					case controlFlowSignalReturn:
						position = nextIndex
						return output.String(), position, controlFlowSignalReturn, nil
					}
				}
			default:
				hasItems = false
			}
		}
	}

loopDone:
	if !hasItems {
		position = skipWhitespace(templateString, nextIndex)
		if matchesDirectiveAt(templateString, position, "else") {
			position = skipWhitespace(templateString, position+len("@else"))
			if position >= len(templateString) || templateString[position] != '{' {
				return "", 0, controlFlowSignalNone, fmt.Errorf("GoSSR malformed @else directive for @for")
			}
			fallbackBody, fallbackNextIndex, err := consumeBalancedBlock(templateString, position)
			if err != nil {
				return "", 0, controlFlowSignalNone, err
			}
			fallbackHtml, signal, err := renderControlFlowFragment(fallbackBody, scopeArguments, locals, depth)
			if err != nil {
				return "", 0, controlFlowSignalNone, err
			}
			output.WriteString(fallbackHtml)
			position = fallbackNextIndex
			if signal != controlFlowSignalNone {
				return output.String(), position, signal, nil
			}
		}
	} else {
		position = nextIndex
	}

	return output.String(), position, controlFlowSignalNone, nil
}

var forHeaderKeywordRegex = regexp.MustCompile(`\bin\b`)

func parseForHeader(header string) (string, string, string, error) {
	loc := forHeaderKeywordRegex.FindStringIndex(header)
	if loc == nil {
		return "", "", "", fmt.Errorf("GoSSR malformed @for header")
	}

	left := strings.TrimSpace(header[:loc[0]])
	rangeExpression := strings.TrimSpace(header[loc[1]:])
	if left == "" || rangeExpression == "" {
		return "", "", "", fmt.Errorf("GoSSR malformed @for header")
	}

	loopIndexName := "index"
	loopValueName := "value"
	if strings.Contains(left, ",") {
		varNameParts := strings.SplitN(left, ",", 2)
		loopIndexName = strings.TrimSpace(varNameParts[0])
		loopValueName = strings.TrimSpace(varNameParts[1])
	} else {
		loopValueName = strings.TrimSpace(left)
	}

	if loopValueName == "" {
		return "", "", "", fmt.Errorf("GoSSR malformed @for header")
	}

	return loopIndexName, loopValueName, rangeExpression, nil
}

type switchCase struct {
	expressions []string
	body        string
	isDefault   bool
}

func renderSwitchDirective(templateString string, startIndex int, scopeArguments []any, locals map[string]any, depth int) (string, int, controlFlowSignal, error) {
	position := skipWhitespace(templateString, startIndex+len("@switch"))
	headEnd := strings.IndexByte(templateString[position:], '{')
	if headEnd == -1 {
		return "", 0, controlFlowSignalNone, fmt.Errorf("GoSSR malformed @switch directive")
	}
	header := strings.TrimSpace(templateString[position : position+headEnd])
	body, nextIndex, err := consumeBalancedBlock(templateString, position+headEnd)
	if err != nil {
		return "", 0, controlFlowSignalNone, err
	}

	switchValueExpression, typeSwitchMode, truthSwitchMode := parseSwitchHeader(header)
	switchValue, found, err := evaluateControlExpression(switchValueExpression, scopeArguments, locals, false)
	if err != nil {
		return "", 0, controlFlowSignalNone, err
	}

	cases, defaultBody, err := parseSwitchCases(body)
	if err != nil {
		return "", 0, controlFlowSignalNone, err
	}

	for _, currentCase := range cases {
		caseMatched := false
		for _, expression := range currentCase.expressions {
			if truthSwitchMode {
				caseValue, ok, err := evaluateControlExpression(expression, scopeArguments, locals, false)
				if err != nil {
					return "", 0, controlFlowSignalNone, err
				}
				if ok && evaluateTruthiness(caseValue) {
					caseMatched = true
					break
				}
				continue
			}

			if typeSwitchMode {
				matched, err := matchesTypeSwitchCase(switchValue, expression, scopeArguments, locals)
				if err != nil {
					return "", 0, controlFlowSignalNone, err
				}
				if matched {
					caseMatched = true
					break
				}
				continue
			}

			caseValue, ok, err := evaluateControlExpression(expression, scopeArguments, locals, true)
			if err != nil {
				return "", 0, controlFlowSignalNone, err
			}
			if !ok && expression != "" {
				caseValue = expression
				ok = true
			}
			if !found {
				continue
			}
			if compareControlValues(switchValue, caseValue) {
				caseMatched = true
				break
			}
		}

		if caseMatched {
			caseHtml, signal, err := renderControlFlowFragment(currentCase.body, scopeArguments, locals, depth)
			if err != nil {
				return "", 0, controlFlowSignalNone, err
			}
			return caseHtml, nextIndex, signal, nil
		}
	}

	if defaultBody != "" {
		defaultHtml, signal, err := renderControlFlowFragment(defaultBody, scopeArguments, locals, depth)
		if err != nil {
			return "", 0, controlFlowSignalNone, err
		}
		return defaultHtml, nextIndex, signal, nil
	}

	return "", nextIndex, controlFlowSignalNone, nil
}

func parseSwitchHeader(header string) (string, bool, bool) {
	trimmed := strings.TrimSpace(header)
	if trimmed == "" {
		return "", false, true
	}
	if strings.Contains(trimmed, ".(type)") {
		before, _, _ := strings.Cut(trimmed, ".(type)")
		if left, right, found := strings.Cut(before, ":="); found {
			_ = left
			before = right
		}
		return strings.TrimSpace(before), true, false
	}
	return trimmed, false, false
}

func parseSwitchCases(body string) ([]switchCase, string, error) {
	var cases []switchCase
	defaultBody := ""

	position := 0
	currentStart := -1
	currentCase := switchCase{}
	braceDepth := 0
	quoteChar := byte(0)

	closeCurrentCase := func(endIndex int) {
		if currentStart == -1 {
			return
		}
		caseBody := strings.TrimSpace(body[currentStart:endIndex])
		if currentCase.isDefault {
			defaultBody = caseBody
		} else {
			currentCase.body = caseBody
			cases = append(cases, currentCase)
		}
		currentStart = -1
		currentCase = switchCase{}
	}

	for position < len(body) {
		if quoteChar != 0 {
			if body[position] == quoteChar {
				quoteChar = 0
			}
			position++
			continue
		}

		switch body[position] {
		case '"', '\'', '`':
			quoteChar = body[position]
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '@':
			if braceDepth == 0 && matchesDirectiveAt(body, position, "case") {
				closeCurrentCase(position)
				position += len("@case")
				position = skipWhitespace(body, position)
				colonIndex := strings.IndexByte(body[position:], ':')
				if colonIndex == -1 {
					return nil, "", fmt.Errorf("GoSSR malformed @case directive")
				}
				expressionString := strings.TrimSpace(body[position : position+colonIndex])
				currentCase = switchCase{expressions: splitSwitchCaseExpressions(expressionString)}
				currentStart = position + colonIndex + 1
				position = currentStart - 1
			} else if braceDepth == 0 && matchesDirectiveAt(body, position, "default") {
				closeCurrentCase(position)
				position += len("@default")
				position = skipWhitespace(body, position)
				if position >= len(body) || body[position] != ':' {
					return nil, "", fmt.Errorf("GoSSR malformed @default directive")
				}
				currentCase = switchCase{isDefault: true}
				currentStart = position + 1
				position = currentStart - 1
			}
		case 'c':
			if braceDepth == 0 && matchesPlainKeywordAt(body, position, "case") {
				closeCurrentCase(position)
				position += len("case")
				position = skipWhitespace(body, position)
				colonIndex := strings.IndexByte(body[position:], ':')
				if colonIndex == -1 {
					return nil, "", fmt.Errorf("GoSSR malformed case directive")
				}
				expressionString := strings.TrimSpace(body[position : position+colonIndex])
				currentCase = switchCase{expressions: splitSwitchCaseExpressions(expressionString)}
				currentStart = position + colonIndex + 1
				position = currentStart - 1
			}
		case 'd':
			if braceDepth == 0 && matchesPlainKeywordAt(body, position, "default") {
				closeCurrentCase(position)
				position += len("default")
				position = skipWhitespace(body, position)
				if position >= len(body) || body[position] != ':' {
					return nil, "", fmt.Errorf("GoSSR malformed default directive")
				}
				currentCase = switchCase{isDefault: true}
				currentStart = position + 1
				position = currentStart - 1
			}
		}
		position++
	}

	closeCurrentCase(len(body))
	return cases, defaultBody, nil
}

func matchesPlainKeywordAt(templateString string, index int, keyword string) bool {
	if index < 0 || index+len(keyword) > len(templateString) {
		return false
	}
	if !strings.HasPrefix(templateString[index:], keyword) {
		return false
	}
	endIndex := index + len(keyword)
	if endIndex < len(templateString) && isDirectiveNameCharacter(templateString[endIndex]) {
		return false
	}
	if index > 0 {
		previous := templateString[index-1]
		if isDirectiveNameCharacter(previous) {
			return false
		}
	}
	return true
}

func splitSwitchCaseExpressions(expressionString string) []string {
	if expressionString == "" {
		return nil
	}

	var expressions []string
	current := strings.Builder{}
	quoteChar := byte(0)
	parenDepth := 0
	for index := 0; index < len(expressionString); index++ {
		currentByte := expressionString[index]
		if quoteChar != 0 {
			if currentByte == quoteChar {
				quoteChar = 0
			}
			current.WriteByte(currentByte)
			continue
		}

		switch currentByte {
		case '"', '\'', '`':
			quoteChar = currentByte
			current.WriteByte(currentByte)
		case '(':
			parenDepth++
			current.WriteByte(currentByte)
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			current.WriteByte(currentByte)
		case ',':
			if parenDepth == 0 {
				expressions = append(expressions, strings.TrimSpace(current.String()))
				current.Reset()
				continue
			}
			current.WriteByte(currentByte)
		default:
			current.WriteByte(currentByte)
		}
	}

	if current.Len() > 0 {
		expressions = append(expressions, strings.TrimSpace(current.String()))
	}

	return expressions
}

func cloneLocals(locals map[string]any) map[string]any {
	cloned := make(map[string]any)
	for key, value := range locals {
		cloned[key] = value
	}
	return cloned
}

func evaluateControlExpression(expressionString string, scopeArguments []any, locals map[string]any, allowLiteralIdent bool) (any, bool, error) {
	trimmed := strings.TrimSpace(expressionString)
	if trimmed == "" {
		return nil, false, nil
	}

	parsedExpression, err := parser.ParseExpr(trimmed)
	if err != nil {
		return nil, false, err
	}

	return evaluateExpressionNode(parsedExpression, scopeArguments, locals, allowLiteralIdent)
}

func evaluateExpressionNode(expression ast.Expr, scopeArguments []any, locals map[string]any, allowLiteralIdent bool) (any, bool, error) {
	switch node := expression.(type) {
	case *ast.BasicLit:
		return parseBasicLiteral(node)
	case *ast.Ident:
		switch node.Name {
		case "true":
			return true, true, nil
		case "false":
			return false, true, nil
		case "nil":
			return nil, true, nil
		}
		if value, found := resolveNamedScopePath(scopeArgumentsWithLocals(scopeArguments, locals), node.Name, nil); found {
			return value, true, nil
		}
		if allowLiteralIdent {
			return node.Name, true, nil
		}
		if isStrict() {
			return nil, false, fmt.Errorf("GoSSR strict mode error: unresolved identifier '%s'", node.Name)
		}
		return nil, false, nil
	case *ast.ParenExpr:
		return evaluateExpressionNode(node.X, scopeArguments, locals, allowLiteralIdent)
	case *ast.SelectorExpr:
		path, ok := collectSelectorPath(node)
		if !ok {
			return nil, false, fmt.Errorf("GoSSR unsupported selector expression")
		}
		if value, found := resolveNamedScopePath(scopeArgumentsWithLocals(scopeArguments, locals), path[0], path[1:]); found {
			return value, true, nil
		}
		if allowLiteralIdent {
			return strings.Join(path, "."), true, nil
		}
		if isStrict() {
			return nil, false, fmt.Errorf("GoSSR strict mode error: unresolved identifier '%s'", strings.Join(path, "."))
		}
		return nil, false, nil
	case *ast.UnaryExpr:
		value, found, err := evaluateExpressionNode(node.X, scopeArguments, locals, allowLiteralIdent)
		if err != nil || !found {
			return value, found, err
		}
		switch node.Op {
		case token.NOT:
			return !evaluateTruthiness(value), true, nil
		case token.ADD:
			return value, true, nil
		case token.SUB:
			return negateValue(value)
		default:
			return nil, false, fmt.Errorf("GoSSR unsupported unary operator %s", node.Op.String())
		}
	case *ast.BinaryExpr:
		if node.Op == token.LAND {
			leftValue, found, err := evaluateExpressionNode(node.X, scopeArguments, locals, allowLiteralIdent)
			if err != nil || !found {
				return leftValue, found, err
			}
			if !evaluateTruthiness(leftValue) {
				return false, true, nil
			}
			rightValue, found, err := evaluateExpressionNode(node.Y, scopeArguments, locals, allowLiteralIdent)
			if err != nil || !found {
				return rightValue, found, err
			}
			return evaluateTruthiness(rightValue), true, nil
		}
		if node.Op == token.LOR {
			leftValue, found, err := evaluateExpressionNode(node.X, scopeArguments, locals, allowLiteralIdent)
			if err != nil || !found {
				return leftValue, found, err
			}
			if evaluateTruthiness(leftValue) {
				return true, true, nil
			}
			rightValue, found, err := evaluateExpressionNode(node.Y, scopeArguments, locals, allowLiteralIdent)
			if err != nil || !found {
				return rightValue, found, err
			}
			return evaluateTruthiness(rightValue), true, nil
		}

		leftValue, leftFound, err := evaluateExpressionNode(node.X, scopeArguments, locals, allowLiteralIdent)
		if err != nil {
			return nil, false, err
		}
		rightValue, rightFound, err := evaluateExpressionNode(node.Y, scopeArguments, locals, allowLiteralIdent)
		if err != nil {
			return nil, false, err
		}
		if !leftFound || !rightFound {
			if isStrict() {
				return nil, false, fmt.Errorf("GoSSR strict mode error: unresolved expression '%s'", trimExpressionForError(expression))
			}
			return false, true, nil
		}
		return compareExpressionValues(leftValue, rightValue, node.Op)
	case *ast.CallExpr:
		return evaluateCallExpression(node, scopeArguments, locals, allowLiteralIdent)
	default:
		return nil, false, fmt.Errorf("GoSSR unsupported control flow expression")
	}
}

func scopeArgumentsWithLocals(scopeArguments []any, locals map[string]any) []any {
	if len(locals) == 0 {
		return scopeArguments
	}
	combined := make([]any, 0, len(scopeArguments)+1)
	combined = append(combined, locals)
	combined = append(combined, scopeArguments...)
	return combined
}

func parseBasicLiteral(literal *ast.BasicLit) (any, bool, error) {
	switch literal.Kind {
	case token.STRING:
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			return nil, false, err
		}
		return value, true, nil
	case token.INT:
		value, err := strconv.ParseInt(literal.Value, 0, 64)
		if err != nil {
			return nil, false, err
		}
		return value, true, nil
	case token.FLOAT:
		value, err := strconv.ParseFloat(literal.Value, 64)
		if err != nil {
			return nil, false, err
		}
		return value, true, nil
	case token.CHAR:
		value, err := strconv.Unquote(literal.Value)
		if err != nil || value == "" {
			return nil, false, err
		}
		return rune(value[0]), true, nil
	default:
		return nil, false, fmt.Errorf("GoSSR unsupported literal %s", literal.Value)
	}
}

func negateValue(value any) (any, bool, error) {
	v := reflect.ValueOf(value)
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return nil, false, nil
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return nil, false, nil
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return -v.Int(), true, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return -int64(v.Uint()), true, nil
	case reflect.Float32, reflect.Float64:
		return -v.Float(), true, nil
	default:
		return nil, false, fmt.Errorf("GoSSR unsupported unary minus operand")
	}
}

func evaluateCallExpression(call *ast.CallExpr, scopeArguments []any, locals map[string]any, allowLiteralIdent bool) (any, bool, error) {
	if identifier, ok := call.Fun.(*ast.Ident); ok && identifier.Name == "len" && len(call.Args) == 1 {
		value, found, err := evaluateExpressionNode(call.Args[0], scopeArguments, locals, allowLiteralIdent)
		if err != nil || !found {
			return value, found, err
		}
		reflectionValue := reflect.ValueOf(value)
		for reflectionValue.IsValid() && (reflectionValue.Kind() == reflect.Pointer || reflectionValue.Kind() == reflect.Interface) {
			if reflectionValue.IsNil() {
				return 0, true, nil
			}
			reflectionValue = reflectionValue.Elem()
		}
		if !reflectionValue.IsValid() {
			return 0, true, nil
		}
		switch reflectionValue.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
			return reflectionValue.Len(), true, nil
		default:
			return 0, true, nil
		}
	}
	return nil, false, fmt.Errorf("GoSSR unsupported function call in control flow expression")
}

func compareExpressionValues(left any, right any, operator token.Token) (any, bool, error) {
	if leftNumeric, leftOk := numericValue(left); leftOk {
		if rightNumeric, rightOk := numericValue(right); rightOk {
			switch operator {
			case token.EQL:
				return leftNumeric == rightNumeric, true, nil
			case token.NEQ:
				return leftNumeric != rightNumeric, true, nil
			case token.GTR:
				return leftNumeric > rightNumeric, true, nil
			case token.GEQ:
				return leftNumeric >= rightNumeric, true, nil
			case token.LSS:
				return leftNumeric < rightNumeric, true, nil
			case token.LEQ:
				return leftNumeric <= rightNumeric, true, nil
			}
		}
	}

	leftString := fmt.Sprintf("%v", left)
	rightString := fmt.Sprintf("%v", right)
	switch operator {
	case token.EQL:
		return leftString == rightString, true, nil
	case token.NEQ:
		return leftString != rightString, true, nil
	case token.GTR:
		return leftString > rightString, true, nil
	case token.GEQ:
		return leftString >= rightString, true, nil
	case token.LSS:
		return leftString < rightString, true, nil
	case token.LEQ:
		return leftString <= rightString, true, nil
	default:
		return nil, false, fmt.Errorf("GoSSR unsupported comparison operator %s", operator.String())
	}
}

func numericValue(value any) (float64, bool) {
	reflectionValue := reflect.ValueOf(value)
	for reflectionValue.IsValid() && (reflectionValue.Kind() == reflect.Pointer || reflectionValue.Kind() == reflect.Interface) {
		if reflectionValue.IsNil() {
			return 0, false
		}
		reflectionValue = reflectionValue.Elem()
	}
	if !reflectionValue.IsValid() {
		return 0, false
	}

	switch reflectionValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(reflectionValue.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(reflectionValue.Uint()), true
	case reflect.Float32, reflect.Float64:
		return reflectionValue.Float(), true
	default:
		return 0, false
	}
}

func compareControlValues(left any, right any) bool {
	if left == nil || right == nil {
		return left == right
	}
	if leftNumeric, leftOk := numericValue(left); leftOk {
		if rightNumeric, rightOk := numericValue(right); rightOk {
			return leftNumeric == rightNumeric
		}
	}
	return reflect.DeepEqual(left, right) || fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)
}

func matchesTypeSwitchCase(value any, caseExpression string, scopeArguments []any, locals map[string]any) (bool, error) {
	if value == nil {
		return false, nil
	}
	reflectionType := reflect.TypeOf(value)
	for reflectionType.Kind() == reflect.Pointer || reflectionType.Kind() == reflect.Interface {
		if reflectionType.Kind() == reflect.Pointer && reflectionType.Elem() == nil {
			return false, nil
		}
		reflectionType = reflectionType.Elem()
	}
	if reflectionType == nil {
		return false, nil
	}

	expressionValue, found, err := evaluateControlExpression(caseExpression, scopeArguments, locals, true)
	if err != nil {
		return false, err
	}
	if !found {
		expressionValue = strings.TrimSpace(caseExpression)
	}
	caseName := fmt.Sprintf("%v", expressionValue)
	return caseName == reflectionType.Name() || caseName == reflectionType.String() || caseName == reflectionType.Kind().String(), nil
}

func trimExpressionForError(expression ast.Expr) string {
	return strings.TrimSpace(fmt.Sprintf("%T", expression))
}

func collectSelectorPath(expression *ast.SelectorExpr) ([]string, bool) {
	if expression == nil {
		return nil, false
	}

	if identifier, ok := expression.X.(*ast.Ident); ok {
		return []string{identifier.Name, expression.Sel.Name}, true
	}

	if selector, ok := expression.X.(*ast.SelectorExpr); ok {
		path, ok := collectSelectorPath(selector)
		if !ok {
			return nil, false
		}
		return append(path, expression.Sel.Name), true
	}

	return nil, false
}
