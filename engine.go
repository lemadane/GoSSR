package gossr

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// SSR represents any component capable of writing HTML directly to an output stream.
type SSR interface {
	Render(writer io.Writer) error
	String() string
}

// EscapeHTML escapes special HTML characters to prevent XSS.
func EscapeHTML(input string) string {
	return html.EscapeString(input)
}

// UnescapeHTML unescapes standard HTML entities to raw text.
func UnescapeHTML(input string) string {
	return html.UnescapeString(input)
}

// RenderHTTP sets the Content-Type header to text/html; charset=utf-8 if not already set,
// and renders the SSR component directly to the http.ResponseWriter.
func RenderHTTP(responseWriter http.ResponseWriter, component SSR) error {
	if component == nil {
		return fmt.Errorf("cannot render nil SSR component")
	}
	if responseWriter.Header().Get("Content-Type") == "" {
		responseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	return component.Render(responseWriter)
}

// ErrorHandler is invoked when RenderHTTP fails inside Handler.
// It defaults to writing HTTP 500 Internal Server Error.
var ErrorHandler = func(responseWriter http.ResponseWriter, request *http.Request, err error) {
	if responseWriter != nil {
		http.Error(responseWriter, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Handler constructs an http.HandlerFunc from a component factory function.
func Handler(factory func(request *http.Request) SSR) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		componentInstance := factory(request)
		if componentInstance != nil {
			err := RenderHTTP(responseWriter, componentInstance)
			if err != nil && ErrorHandler != nil {
				ErrorHandler(responseWriter, request, err)
			}
		}
	}
}

// MaxRenderDepth defines the maximum allowed component nesting depth before throwing a recursion error.
const MaxRenderDepth = 100

var strictModeAtomic int32

// SetStrict enables or disables strict mode property resolution error checking in a thread-safe manner.
func SetStrict(strict bool) {
	if strict {
		atomic.StoreInt32(&strictModeAtomic, 1)
	} else {
		atomic.StoreInt32(&strictModeAtomic, 0)
	}
}

// Strict enables or disables strict mode property resolution error checking.
func Strict(strict bool) {
	SetStrict(strict)
}

func isStrict() bool {
	return atomic.LoadInt32(&strictModeAtomic) == 1
}

var (
	mapRegex                    = regexp.MustCompile(`\$\{properties\.([a-zA-Z0-9_\.]+)\.map\(([a-zA-Z0-9_]+)\s*=>\s*(.*?)\)\}`)
	propertyRegex               = regexp.MustCompile(`\$\{properties\.([a-zA-Z0-9_\.]+)\}`)
	genericItemPropertyRegex    = regexp.MustCompile(`\$\{([a-zA-Z0-9_]+)(?:\.([a-zA-Z0-9_\.]+))?\}`)
	genericQuotedTernaryRegex   = regexp.MustCompile(`\$\{([a-zA-Z0-9_]+)\.([a-zA-Z0-9_\.]+)\s*(?:(==|!=)\s*["']?([^"'\s\?]+)["']?)?\s*\?\s*"([^"]*)"\s*:\s*"([^"]*)"\}`)
	genericUnquotedTernaryRegex = regexp.MustCompile(`\$\{([a-zA-Z0-9_]+)\.([a-zA-Z0-9_\.]+)\s*(?:(==|!=)\s*["']?([^"'\s\?]+)["']?)?\s*\?\s*([^:\s\}]+)\s*:\s*([^}\s]+)\}`)
)

// RawHtml represents trusted unescaped HTML content that should not be HTML-escaped during rendering.
type RawHtml struct {
	Value string
}

// Raw creates a RawHtml wrapper for trusted raw HTML content.
func Raw(value string) RawHtml {
	return RawHtml{Value: value}
}

// RawHtmlOf creates a RawHtml wrapper for trusted raw HTML content.
func RawHtmlOf(value string) RawHtml {
	return RawHtml{Value: value}
}

func (rawHtml RawHtml) Render(writer io.Writer) error {
	_, err := io.WriteString(writer, rawHtml.Value)
	return err
}

func (rawHtml RawHtml) String() string {
	return rawHtml.Value
}

// SafeUrl represents a sanitized URL wrapper that enforces a strict protocol allowlist
// (http, https, mailto, tel, relative paths) and blocks dangerous protocols like javascript:, vbscript:, and data:.
type SafeUrl struct {
	Value string
}

// URL creates a SafeUrl wrapper for sanitizing URL attributes.
func URL(value string) SafeUrl {
	return SafeUrl{Value: value}
}

// SafeUrlOf creates a SafeUrl wrapper for sanitizing URL attributes.
func SafeUrlOf(value string) SafeUrl {
	return SafeUrl{Value: value}
}

// SanitizeUrl sanitizes a URL string using a strict protocol allowlist.
func SanitizeUrl(url string) string {
	if url == "" || strings.TrimSpace(url) == "" {
		return ""
	}

	raw := strings.TrimSpace(url)

	// Unescape HTML entities before protocol checking to prevent entity bypasses (e.g. java&#115;cript:)
	unescaped := strings.ToLower(strings.TrimSpace(html.UnescapeString(raw)))

	// Strip non-printable ASCII / control characters
	var cleanScheme strings.Builder
	for charIndex := 0; charIndex < len(unescaped); charIndex++ {
		charByte := unescaped[charIndex]
		if charByte > 32 && charByte < 127 {
			cleanScheme.WriteByte(charByte)
		}
	}
	normalized := cleanScheme.String()

	// Allow relative URLs, fragment IDs, query strings, and absolute path URLs
	if strings.HasPrefix(normalized, "/") || strings.HasPrefix(normalized, "#") || strings.HasPrefix(normalized, "?") || strings.HasPrefix(normalized, ".") {
		return raw
	}

	colonIndex := strings.Index(normalized, ":")
	if colonIndex == -1 {
		// No scheme specified; treat as relative URL path
		return raw
	}

	scheme := normalized[:colonIndex]
	if scheme == "http" || scheme == "https" || scheme == "mailto" || scheme == "tel" {
		return raw
	}

	// Unsafe or unrecognized protocol scheme
	return "about:blank"
}

func (safeUrl SafeUrl) Render(writer io.Writer) error {
	_, err := io.WriteString(writer, html.EscapeString(SanitizeUrl(safeUrl.Value)))
	return err
}

func (safeUrl SafeUrl) String() string {
	return html.EscapeString(SanitizeUrl(safeUrl.Value))
}

type renderComponent struct {
	templateString string
	scopeArguments []any
	depth          int
}

func (component renderComponent) String() string {
	var stringBuilder strings.Builder
	err := component.Render(&stringBuilder)
	if err != nil {
		return fmt.Sprintf("<!-- Render Error: %v -->", err)
	}
	return stringBuilder.String()
}

func (component renderComponent) Render(writer io.Writer) error {
	if component.depth > MaxRenderDepth {
		return fmt.Errorf("GoSSR component recursion limit exceeded (max depth %d)", MaxRenderDepth)
	}

	if err := validateInterpolationSecurityContext(component.templateString); err != nil {
		return err
	}

	outputHtml := component.templateString

	for _, argument := range component.scopeArguments {
		var err error
		outputHtml, err = processScopeArgumentProperties(outputHtml, argument)
		if err != nil {
			return err
		}
	}

	processedHtml, err := processCustomComponentTags(outputHtml, component.depth)
	if err != nil {
		return err
	}

	_, writeError := io.WriteString(writer, processedHtml)
	return writeError
}

// Render constructs a new SSR component by binding a backtick template string to scope property structs.
func Render(templateString string, scopeArguments ...any) SSR {
	return renderComponent{
		templateString: templateString,
		scopeArguments: scopeArguments,
		depth:          0,
	}
}

// CompiledTemplate represents a pre-parsed, immutable template AST ready for fast binding.
type CompiledTemplate struct {
	rawTemplate string
	astNodes    []templateASTNode
}

type templateASTNode interface {
	renderAST(writer io.Writer, scopeArguments []any, depth int, strict bool) error
}

type astStaticTextNode struct {
	text string
}

func (n astStaticTextNode) renderAST(writer io.Writer, scopeArguments []any, depth int, strict bool) error {
	_, err := io.WriteString(writer, n.text)
	return err
}

type astPropertyNode struct {
	prefix      string
	path        []string
	pathString  string
	attrContext string
}

func (n astPropertyNode) renderAST(writer io.Writer, scopeArguments []any, depth int, strict bool) error {
	var value any
	var found bool

	if len(n.path) == 0 {
		for _, arg := range scopeArguments {
			if arg != nil {
				value = arg
				found = true
				break
			}
		}
	} else {
		for _, arg := range scopeArguments {
			value, found = resolvePropertyPath(arg, n.path)
			if found {
				break
			}
		}
	}

	if !found {
		if strict {
			prefix := n.prefix
			if prefix == "" {
				prefix = "properties"
			}
			return fmt.Errorf("GoSSR strict mode error: unresolved property path '${%s.%s}'", prefix, n.pathString)
		}
		if len(n.path) == 0 {
			_, err := io.WriteString(writer, fmt.Sprintf("${%s}", n.prefix))
			return err
		}
		_, err := io.WriteString(writer, fmt.Sprintf("${%s.%s}", n.prefix, n.pathString))
		return err
	}
	return formatAndRenderValueForContext(writer, value, n.attrContext)
}

type astTernaryNode struct {
	prefix      string
	path        []string
	pathString  string
	operator    string
	targetValue string
	trueBranch  string
	falseBranch string
}

func (n astTernaryNode) renderAST(writer io.Writer, scopeArguments []any, depth int, strict bool) error {
	var value any
	var found bool
	for _, arg := range scopeArguments {
		value, found = resolvePropertyPath(arg, n.path)
		if found {
			break
		}
	}

	if !found {
		if strict {
			return fmt.Errorf("GoSSR strict mode error: unresolved ternary property path '${%s.%s}'", n.prefix, n.pathString)
		}
		_, err := io.WriteString(writer, n.falseBranch)
		return err
	}

	var conditionMet bool
	if n.operator == "==" {
		conditionMet = fmt.Sprintf("%v", value) == n.targetValue
	} else if n.operator == "!=" {
		conditionMet = fmt.Sprintf("%v", value) != n.targetValue
	} else {
		conditionMet = evaluateTruthiness(value)
	}

	resultText := n.falseBranch
	if conditionMet {
		resultText = n.trueBranch
	}
	_, err := io.WriteString(writer, resultText)
	return err
}

type astMapNode struct {
	path       []string
	pathString string
	varName    string
	bodyAST    []templateASTNode
}

func (n astMapNode) renderAST(writer io.Writer, scopeArguments []any, depth int, strict bool) error {
	var targetValue any
	var found bool
	for _, arg := range scopeArguments {
		targetValue, found = resolvePropertyPath(arg, n.path)
		if found {
			break
		}
	}

	if !found {
		if strict {
			return fmt.Errorf("GoSSR strict mode error: unresolved map property path '${properties.%s}'", n.pathString)
		}
		return nil
	}

	sliceValue := reflect.ValueOf(targetValue)
	if sliceValue.Kind() == reflect.Pointer {
		sliceValue = sliceValue.Elem()
	}
	if sliceValue.Kind() != reflect.Slice && sliceValue.Kind() != reflect.Array {
		return nil
	}

	for itemIndex := 0; itemIndex < sliceValue.Len(); itemIndex++ {
		item := sliceValue.Index(itemIndex).Interface()
		for _, childNode := range n.bodyAST {
			if err := childNode.renderAST(writer, []any{item}, depth, strict); err != nil {
				return err
			}
		}
	}
	return nil
}

type astCustomTagNode struct {
	rawTagText string
}

func (n astCustomTagNode) renderAST(writer io.Writer, scopeArguments []any, depth int, strict bool) error {
	expandedHtml, err := processCustomComponentTags(n.rawTagText, depth)
	if err != nil {
		return err
	}
	_, err = io.WriteString(writer, expandedHtml)
	return err
}

func parseTemplateAST(templateString string) []templateASTNode {
	var nodes []templateASTNode

	matches := mapRegex.FindAllStringSubmatchIndex(templateString, -1)
	if len(matches) > 0 {
		lastIndex := 0
		for _, loc := range matches {
			matchStart, matchEnd := loc[0], loc[1]
			if matchStart > lastIndex {
				nodes = append(nodes, parseFlatTemplateAST(templateString[lastIndex:matchStart])...)
			}

			pathString := templateString[loc[2]:loc[3]]
			variableName := templateString[loc[4]:loc[5]]
			bodyExpression := templateString[loc[6]:loc[7]]

			nodes = append(nodes, astMapNode{
				path:       strings.Split(pathString, "."),
				pathString: pathString,
				varName:    variableName,
				bodyAST:    parseTemplateAST(bodyExpression),
			})
			lastIndex = matchEnd
		}
		if lastIndex < len(templateString) {
			nodes = append(nodes, parseFlatTemplateAST(templateString[lastIndex:])...)
		}
		return nodes
	}

	return parseFlatTemplateAST(templateString)
}

func parseFlatTemplateAST(templateString string) []templateASTNode {
	return parseExpressionsOnly(templateString)
}

func parseExpressionsOnly(templateString string) []templateASTNode {
	var nodes []templateASTNode
	var pos int

	for pos < len(templateString) {
		qTernaryLoc := genericQuotedTernaryRegex.FindStringIndex(templateString[pos:])
		uTernaryLoc := genericUnquotedTernaryRegex.FindStringIndex(templateString[pos:])
		propLoc := propertyRegex.FindStringIndex(templateString[pos:])
		itemPropLoc := genericItemPropertyRegex.FindStringIndex(templateString[pos:])

		earliestType := 0
		earliestStart := len(templateString)
		earliestLoc := []int(nil)

		if qTernaryLoc != nil && qTernaryLoc[0] < earliestStart {
			earliestType = 1
			earliestStart = qTernaryLoc[0]
			earliestLoc = qTernaryLoc
		}
		if uTernaryLoc != nil && uTernaryLoc[0] < earliestStart {
			earliestType = 2
			earliestStart = uTernaryLoc[0]
			earliestLoc = uTernaryLoc
		}
		if propLoc != nil && propLoc[0] < earliestStart {
			earliestType = 3
			earliestStart = propLoc[0]
			earliestLoc = propLoc
		}
		if itemPropLoc != nil && itemPropLoc[0] < earliestStart {
			earliestType = 4
			earliestStart = itemPropLoc[0]
			earliestLoc = itemPropLoc
		}

		if earliestType == 0 {
			if pos < len(templateString) {
				nodes = append(nodes, astStaticTextNode{text: templateString[pos:]})
			}
			break
		}

		matchStart := pos + earliestLoc[0]
		matchEnd := pos + earliestLoc[1]

		if matchStart > pos {
			nodes = append(nodes, astStaticTextNode{text: templateString[pos:matchStart]})
		}

		matchStr := templateString[matchStart:matchEnd]
		attrContext := getInterpolationAttributeContext(templateString, matchStart)

		if earliestType == 1 {
			submatches := genericQuotedTernaryRegex.FindStringSubmatch(matchStr)
			if len(submatches) >= 7 {
				nodes = append(nodes, astTernaryNode{
					prefix:      submatches[1],
					path:        strings.Split(submatches[2], "."),
					pathString:  submatches[2],
					operator:    submatches[3],
					targetValue: submatches[4],
					trueBranch:  submatches[5],
					falseBranch: submatches[6],
				})
			}
		} else if earliestType == 2 {
			submatches := genericUnquotedTernaryRegex.FindStringSubmatch(matchStr)
			if len(submatches) >= 7 {
				nodes = append(nodes, astTernaryNode{
					prefix:      submatches[1],
					path:        strings.Split(submatches[2], "."),
					pathString:  submatches[2],
					operator:    submatches[3],
					targetValue: submatches[4],
					trueBranch:  strings.Trim(submatches[5], `"`),
					falseBranch: strings.Trim(submatches[6], `"`),
				})
			}
		} else if earliestType == 3 {
			submatches := propertyRegex.FindStringSubmatch(matchStr)
			if len(submatches) >= 2 {
				nodes = append(nodes, astPropertyNode{
					prefix:      "properties",
					path:        strings.Split(submatches[1], "."),
					pathString:  submatches[1],
					attrContext: attrContext,
				})
			}
		} else if earliestType == 4 {
			submatches := genericItemPropertyRegex.FindStringSubmatch(matchStr)
			if len(submatches) >= 2 {
				var path []string
				var pathStr string
				if len(submatches) >= 3 && submatches[2] != "" {
					path = strings.Split(submatches[2], ".")
					pathStr = submatches[2]
				}
				nodes = append(nodes, astPropertyNode{
					prefix:      submatches[1],
					path:        path,
					pathString:  pathStr,
					attrContext: attrContext,
				})
			}
		}

		pos = matchEnd
	}

	return nodes
}

// Compile pre-validates and parses a template string into a pre-compiled AST.
func Compile(templateString string) (CompiledTemplate, error) {
	if err := validateInterpolationSecurityContext(templateString); err != nil {
		return CompiledTemplate{}, err
	}
	astNodes := parseTemplateAST(templateString)
	return CompiledTemplate{
		rawTemplate: templateString,
		astNodes:    astNodes,
	}, nil
}

// MustCompile compiles a template string into an AST or panics if compilation fails security validation.
func MustCompile(templateString string) CompiledTemplate {
	compiled, err := Compile(templateString)
	if err != nil {
		panic(err)
	}
	return compiled
}

type compiledComponent struct {
	compiledTemplate CompiledTemplate
	scopeArguments   []any
}

func (c compiledComponent) String() string {
	var sb strings.Builder
	if err := c.Render(&sb); err != nil {
		return fmt.Sprintf("<!-- Render Error: %v -->", err)
	}
	return sb.String()
}

func (c compiledComponent) Render(writer io.Writer) error {
	var sb strings.Builder
	for _, node := range c.compiledTemplate.astNodes {
		if err := node.renderAST(&sb, c.scopeArguments, 0, isStrict()); err != nil {
			return err
		}
	}
	expandedHtml, err := processCustomComponentTags(sb.String(), 0)
	if err != nil {
		return err
	}
	_, writeErr := io.WriteString(writer, expandedHtml)
	return writeErr
}

// Bind returns an SSR component bound to the compiled AST and scope arguments.
func (compiledTemplate CompiledTemplate) Bind(scopeArguments ...any) SSR {
	return compiledComponent{
		compiledTemplate: compiledTemplate,
		scopeArguments:   scopeArguments,
	}
}

// Render renders the compiled template directly with scope arguments to an io.Writer.
func (compiledTemplate CompiledTemplate) Render(writer io.Writer, scopeArguments ...any) error {
	for _, node := range compiledTemplate.astNodes {
		if err := node.renderAST(writer, scopeArguments, 0, isStrict()); err != nil {
			return err
		}
	}
	return nil
}

func isUrlAttribute(attrName string) bool {
	lower := strings.ToLower(attrName)
	return lower == "href" || lower == "src" || lower == "action" || lower == "formaction" ||
		lower == "cite" || lower == "data" || lower == "poster" || lower == "icon"
}

func getInterpolationAttributeContext(templateHtml string, matchIndex int) string {
	templateLength := len(templateHtml)
	if matchIndex > templateLength {
		matchIndex = templateLength
	}

	inTag := false
	var quoteChar byte = 0
	currentAttribute := ""
	var attributeBuffer strings.Builder
	blockContext := ""

	for scanIndex := 0; scanIndex < matchIndex; scanIndex++ {
		if blockContext == "comment" {
			if scanIndex+2 < templateLength && strings.HasPrefix(templateHtml[scanIndex:], "-->") {
				scanIndex += 2
				blockContext = ""
				continue
			}
		} else if blockContext == "script" {
			if scanIndex+8 < templateLength && strings.ToLower(templateHtml[scanIndex:scanIndex+9]) == "</script>" {
				scanIndex += 8
				blockContext = ""
				continue
			}
		} else if blockContext == "style" {
			if scanIndex+7 < templateLength && strings.ToLower(templateHtml[scanIndex:scanIndex+8]) == "</style>" {
				scanIndex += 7
				blockContext = ""
				continue
			}
		} else {
			currentChar := templateHtml[scanIndex]
			if currentChar == '<' {
				if scanIndex+3 < templateLength && strings.HasPrefix(templateHtml[scanIndex:], "<!--") {
					blockContext = "comment"
				} else if scanIndex+6 < templateLength && strings.HasPrefix(strings.ToLower(templateHtml[scanIndex:]), "<script") {
					nextByte := byte('>')
					if scanIndex+7 < templateLength {
						nextByte = templateHtml[scanIndex+7]
					}
					if isWhitespaceCharacter(nextByte) || nextByte == '>' {
						blockContext = "script"
					}
				} else if scanIndex+5 < templateLength && strings.HasPrefix(strings.ToLower(templateHtml[scanIndex:]), "<style") {
					nextByte := byte('>')
					if scanIndex+6 < templateLength {
						nextByte = templateHtml[scanIndex+6]
					}
					if isWhitespaceCharacter(nextByte) || nextByte == '>' {
						blockContext = "style"
					}
				}
				inTag = true
				currentAttribute = ""
				attributeBuffer.Reset()
			} else if inTag {
				if quoteChar != 0 {
					if currentChar == quoteChar {
						quoteChar = 0
						currentAttribute = ""
						attributeBuffer.Reset()
					}
				} else {
					if currentChar == '"' || currentChar == '\'' {
						quoteChar = currentChar
						currentAttribute = strings.TrimSpace(attributeBuffer.String())
					} else if currentChar == '=' {
						currentAttribute = strings.TrimSpace(attributeBuffer.String())
					} else if currentChar == '>' {
						inTag = false
						quoteChar = 0
						currentAttribute = ""
						attributeBuffer.Reset()
					} else if isWhitespaceCharacter(currentChar) {
						attributeBuffer.Reset()
					} else {
						attributeBuffer.WriteByte(currentChar)
					}
				}
			}
		}
	}

	if inTag && currentAttribute != "" {
		return currentAttribute
	}
	return ""
}

func formatAndRenderValueForContext(writer io.Writer, value any, attrContext string) error {
	if value == nil {
		return nil
	}

	reflectionValue := reflect.ValueOf(value)
	if (reflectionValue.Kind() == reflect.Pointer || reflectionValue.Kind() == reflect.Interface || reflectionValue.Kind() == reflect.Slice || reflectionValue.Kind() == reflect.Map) && reflectionValue.IsNil() {
		return nil
	}

	if ssr, isSSR := value.(SSR); isSSR {
		if attrContext != "" {
			if isUrlAttribute(attrContext) {
				// Inside URL attributes (href, src, action, etc.), RawHtml/SSR components MUST be URL sanitized
				// unless they are SafeUrl!
				if safeUrl, isSafeUrl := value.(SafeUrl); isSafeUrl {
					_, err := io.WriteString(writer, html.EscapeString(SanitizeUrl(safeUrl.Value)))
					return err
				}
				rawStr := ssr.String()
				_, err := io.WriteString(writer, html.EscapeString(SanitizeUrl(rawStr)))
				return err
			}

			// Normal attribute context (id, class, etc.): escape double quotes
			var sb strings.Builder
			if err := ssr.Render(&sb); err != nil {
				return err
			}
			_, err := io.WriteString(writer, html.EscapeString(sb.String()))
			return err
		}

		// Body context: render child SSR component directly into writer
		return ssr.Render(writer)
	}

	if reflectionValue.Kind() == reflect.Slice || reflectionValue.Kind() == reflect.Array {
		for itemIndex := 0; itemIndex < reflectionValue.Len(); itemIndex++ {
			item := reflectionValue.Index(itemIndex).Interface()
			if err := formatAndRenderValueForContext(writer, item, attrContext); err != nil {
				return err
			}
		}
		return nil
	}

	valStr := fmt.Sprintf("%v", value)
	if isUrlAttribute(attrContext) {
		_, err := io.WriteString(writer, html.EscapeString(SanitizeUrl(valStr)))
		return err
	}

	_, err := io.WriteString(writer, html.EscapeString(valStr))
	return err
}

func formatAndEscapeValueForContext(value any, attrContext string) string {
	var sb strings.Builder
	_ = formatAndRenderValueForContext(&sb, value, attrContext)
	return sb.String()
}

func formatAndEscapeValue(value any) string {
	return formatAndEscapeValueForContext(value, "")
}

func resolvePropertyPath(root any, path []string) (any, bool) {
	if len(path) == 0 || root == nil {
		return nil, false
	}

	current := reflect.ValueOf(root)

	for _, name := range path {
		for current.Kind() == reflect.Pointer || current.Kind() == reflect.Interface {
			if current.IsNil() {
				return nil, true
			}
			current = current.Elem()
		}

		if current.Kind() == reflect.Struct {
			field := current.FieldByName(name)
			if !field.IsValid() || !field.CanInterface() {
				return nil, false
			}
			current = field
		} else if current.Kind() == reflect.Map {
			mapKeyType := current.Type().Key()
			var mapKey reflect.Value
			if mapKeyType.Kind() == reflect.String {
				mapKey = reflect.ValueOf(name).Convert(mapKeyType)
			} else if mapKeyType.Kind() == reflect.Int || mapKeyType.Kind() == reflect.Int64 || mapKeyType.Kind() == reflect.Int32 || mapKeyType.Kind() == reflect.Int16 || mapKeyType.Kind() == reflect.Int8 {
				if intVal, err := strconv.ParseInt(name, 10, 64); err == nil {
					mapKey = reflect.ValueOf(intVal).Convert(mapKeyType)
				} else {
					return nil, false
				}
			} else if mapKeyType.Kind() == reflect.Uint || mapKeyType.Kind() == reflect.Uint64 || mapKeyType.Kind() == reflect.Uint32 || mapKeyType.Kind() == reflect.Uint16 || mapKeyType.Kind() == reflect.Uint8 {
				if uintVal, err := strconv.ParseUint(name, 10, 64); err == nil {
					mapKey = reflect.ValueOf(uintVal).Convert(mapKeyType)
				} else {
					return nil, false
				}
			} else {
				return nil, false
			}

			mapValue := current.MapIndex(mapKey)
			if !mapValue.IsValid() || !mapValue.CanInterface() {
				return nil, false
			}
			current = mapValue
		} else {
			return nil, false
		}
	}

	for current.Kind() == reflect.Pointer || current.Kind() == reflect.Interface {
		if current.IsNil() {
			return nil, true
		}
		current = current.Elem()
	}

	return current.Interface(), true
}

func processScopeArgumentProperties(templateString string, argument any) (string, error) {
	if argument == nil {
		return templateString, nil
	}

	// 1. Process map expressions first: ${properties.<path>.map(<variableName> => <bodyExpression>)}
	matches := mapRegex.FindAllStringSubmatchIndex(templateString, -1)
	if len(matches) > 0 {
		var sb strings.Builder
		lastIndex := 0
		for _, loc := range matches {
			matchStart, matchEnd := loc[0], loc[1]
			sb.WriteString(templateString[lastIndex:matchStart])

			pathString := templateString[loc[2]:loc[3]]
			variableName := templateString[loc[4]:loc[5]]
			bodyExpression := templateString[loc[6]:loc[7]]

			path := strings.Split(pathString, ".")
			targetValue, found := resolvePropertyPath(argument, path)
			if !found {
				if isStrict() {
					return "", fmt.Errorf("GoSSR strict mode error: unresolved map property path '${properties.%s}'", pathString)
				}
				sb.WriteString(templateString[matchStart:matchEnd])
				lastIndex = matchEnd
				continue
			}

			sliceValue := reflect.ValueOf(targetValue)
			if sliceValue.Kind() == reflect.Pointer {
				sliceValue = sliceValue.Elem()
			}
			if sliceValue.Kind() != reflect.Slice && sliceValue.Kind() != reflect.Array {
				sb.WriteString(templateString[matchStart:matchEnd])
				lastIndex = matchEnd
				continue
			}

			var renderedItems strings.Builder
			variablePattern := "${" + variableName

			for itemIndex := 0; itemIndex < sliceValue.Len(); itemIndex++ {
				item := sliceValue.Index(itemIndex).Interface()

				if strings.Contains(bodyExpression, variablePattern) {
					itemHtml, err := processMapItemTemplate(bodyExpression, variableName, item)
					if err != nil {
						return "", err
					}
					renderedItems.WriteString(itemHtml)
				} else {
					if renderableItem, isRenderable := item.(SSR); isRenderable {
						renderedItems.WriteString(renderableItem.String())
					} else if stringerItem, isStringer := item.(fmt.Stringer); isStringer {
						attrContext := getInterpolationAttributeContext(templateString, matchStart)
						if isUrlAttribute(attrContext) {
							renderedItems.WriteString(html.EscapeString(SanitizeUrl(stringerItem.String())))
						} else {
							renderedItems.WriteString(html.EscapeString(stringerItem.String()))
						}
					} else {
						attrContext := getInterpolationAttributeContext(templateString, matchStart)
						renderedItems.WriteString(formatAndEscapeValueForContext(item, attrContext))
					}
				}
			}

			sb.WriteString(renderedItems.String())
			lastIndex = matchEnd
		}
		sb.WriteString(templateString[lastIndex:])
		templateString = sb.String()
	}

	// 2. Process ternary expressions: ${properties.<path> ? "A" : "B"} or equality ${properties.<path> == "val" ? "A" : "B"}
	var err error
	templateString, err = processTernaryExpressions(templateString, "properties", argument)
	if err != nil {
		return "", err
	}

	// 3. Process direct property placeholders: ${properties.<path>}
	propMatches := propertyRegex.FindAllStringSubmatchIndex(templateString, -1)
	if len(propMatches) > 0 {
		var sb strings.Builder
		lastIndex := 0
		for _, loc := range propMatches {
			matchStart, matchEnd := loc[0], loc[1]
			sb.WriteString(templateString[lastIndex:matchStart])

			pathString := templateString[loc[2]:loc[3]]
			path := strings.Split(pathString, ".")

			value, found := resolvePropertyPath(argument, path)
			if found {
				attrContext := getInterpolationAttributeContext(templateString, matchStart)
				if err := formatAndRenderValueForContext(&sb, value, attrContext); err != nil {
					return "", err
				}
			} else {
				if isStrict() {
					return "", fmt.Errorf("GoSSR strict mode error: unresolved property path '${properties.%s}'", pathString)
				}
				sb.WriteString(templateString[matchStart:matchEnd])
			}
			lastIndex = matchEnd
		}
		sb.WriteString(templateString[lastIndex:])
		templateString = sb.String()
	}

	return templateString, nil
}

func processMapItemTemplate(templateString string, variableName string, item any) (string, error) {
	if err := validateInterpolationSecurityContext(templateString); err != nil {
		return "", err
	}

	// Process ternary expressions on variableName: ${variableName.<path> ? "A" : "B"}
	var err error
	templateString, err = processTernaryExpressions(templateString, variableName, item)
	if err != nil {
		return "", err
	}

	// Process item property placeholders: ${variableName.<path>} and ${variableName}
	matches := genericItemPropertyRegex.FindAllStringSubmatchIndex(templateString, -1)
	if len(matches) > 0 {
		var sb strings.Builder
		lastIndex := 0
		for _, loc := range matches {
			matchStart, matchEnd := loc[0], loc[1]

			varName := templateString[loc[2]:loc[3]]
			if varName != variableName {
				continue
			}

			sb.WriteString(templateString[lastIndex:matchStart])

			if loc[4] == -1 || loc[5] == -1 {
				// Direct reference ${variableName}
				attrContext := getInterpolationAttributeContext(templateString, matchStart)
				if err := formatAndRenderValueForContext(&sb, item, attrContext); err != nil {
					return "", err
				}
			} else {
				pathString := templateString[loc[4]:loc[5]]
				path := strings.Split(pathString, ".")

				value, found := resolvePropertyPath(item, path)
				if found {
					attrContext := getInterpolationAttributeContext(templateString, matchStart)
					if err := formatAndRenderValueForContext(&sb, value, attrContext); err != nil {
						return "", err
					}
				} else {
					if isStrict() {
						return "", fmt.Errorf("GoSSR strict mode error: unresolved property path '${%s.%s}'", variableName, pathString)
					}
					sb.WriteString(templateString[matchStart:matchEnd])
				}
			}
			lastIndex = matchEnd
		}
		sb.WriteString(templateString[lastIndex:])
		templateString = sb.String()
	}

	return templateString, nil
}

func processTernaryExpressions(templateString string, prefix string, scope any) (string, error) {
	var strictErr error

	// Quoted ternary with optional comparison operator: ${prefix.path == "val" ? "A" : "B"} or ${prefix.path ? "A" : "B"}
	templateString = genericQuotedTernaryRegex.ReplaceAllStringFunc(templateString, func(match string) string {
		submatches := genericQuotedTernaryRegex.FindStringSubmatch(match)
		if len(submatches) >= 7 && submatches[1] == prefix {
			path := strings.Split(submatches[2], ".")
			operator := submatches[3]
			targetValue := submatches[4]
			trueBranch := submatches[5]
			falseBranch := submatches[6]

			value, found := resolvePropertyPath(scope, path)
			if !found {
				if isStrict() && strictErr == nil {
					strictErr = fmt.Errorf("GoSSR strict mode error: unresolved ternary property path '${%s.%s}'", prefix, submatches[2])
				}
				return match
			}

			var conditionMet bool
			if operator == "==" {
				conditionMet = fmt.Sprintf("%v", value) == targetValue
			} else if operator == "!=" {
				conditionMet = fmt.Sprintf("%v", value) != targetValue
			} else {
				conditionMet = evaluateTruthiness(value)
			}

			if conditionMet {
				return trueBranch
			}
			return falseBranch
		}
		return match
	})

	if strictErr != nil {
		return "", strictErr
	}

	// Unquoted ternary
	templateString = genericUnquotedTernaryRegex.ReplaceAllStringFunc(templateString, func(match string) string {
		submatches := genericUnquotedTernaryRegex.FindStringSubmatch(match)
		if len(submatches) >= 7 && submatches[1] == prefix {
			path := strings.Split(submatches[2], ".")
			operator := submatches[3]
			targetValue := submatches[4]
			trueBranch := strings.Trim(submatches[5], `"`)
			falseBranch := strings.Trim(submatches[6], `"`)

			value, found := resolvePropertyPath(scope, path)
			if !found {
				if isStrict() && strictErr == nil {
					strictErr = fmt.Errorf("GoSSR strict mode error: unresolved ternary property path '${%s.%s}'", prefix, submatches[2])
				}
				return match
			}

			var conditionMet bool
			if operator == "==" {
				conditionMet = fmt.Sprintf("%v", value) == targetValue
			} else if operator == "!=" {
				conditionMet = fmt.Sprintf("%v", value) != targetValue
			} else {
				conditionMet = evaluateTruthiness(value)
			}

			if conditionMet {
				return trueBranch
			}
			return falseBranch
		}
		return match
	})

	if strictErr != nil {
		return "", strictErr
	}

	return templateString, nil
}

func evaluateTruthiness(value any) bool {
	if value == nil {
		return false
	}
	reflectionValue := reflect.ValueOf(value)
	for reflectionValue.Kind() == reflect.Pointer || reflectionValue.Kind() == reflect.Interface {
		if reflectionValue.IsNil() {
			return false
		}
		reflectionValue = reflectionValue.Elem()
	}
	switch reflectionValue.Kind() {
	case reflect.Bool:
		return reflectionValue.Bool()
	case reflect.String:
		return reflectionValue.String() != ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflectionValue.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return reflectionValue.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return reflectionValue.Float() != 0
	case reflect.Slice, reflect.Map, reflect.Array:
		return reflectionValue.Len() > 0
	default:
		return true
	}
}

func isExecutableAttribute(lowerAttr string) bool {
	if strings.HasPrefix(lowerAttr, "on") || lowerAttr == "style" {
		return true
	}
	// Alpine.js directives
	if lowerAttr == "x-data" || lowerAttr == "x-init" || lowerAttr == "x-effect" ||
		lowerAttr == "x-show" || lowerAttr == "x-text" || lowerAttr == "x-html" ||
		lowerAttr == "x-if" || lowerAttr == "x-for" || lowerAttr == "x-model" ||
		strings.HasPrefix(lowerAttr, "x-on:") || strings.HasPrefix(lowerAttr, "@") ||
		strings.HasPrefix(lowerAttr, "x-bind:") || (strings.HasPrefix(lowerAttr, ":") && lowerAttr != ":") {
		return true
	}
	// HTMX executable attributes
	if strings.HasPrefix(lowerAttr, "hx-on:") || strings.HasPrefix(lowerAttr, "hx-on::") ||
		lowerAttr == "hx-vals" || lowerAttr == "hx-headers" || lowerAttr == "hx-vars" {
		return true
	}
	return false
}

func validateInterpolationSecurityContext(templateHtml string) error {
	templateHtmlLength := len(templateHtml)
	scanIndex := 0

	inTag := false
	var quoteChar byte = 0
	blockContext := "" // "script", "style", "comment"
	currentAttribute := ""
	var attributeBuffer strings.Builder

	for scanIndex < templateHtmlLength {
		if scanIndex+1 < templateHtmlLength && templateHtml[scanIndex] == '$' && templateHtml[scanIndex+1] == '{' {
			end := strings.IndexByte(templateHtml[scanIndex+2:], '}')
			variableName := "..."
			if end != -1 {
				endPos := scanIndex + 2 + end
				variableName = strings.TrimSpace(templateHtml[scanIndex+2 : endPos])
			}
			if blockContext != "" {
				if blockContext == "script" {
					return fmt.Errorf("GoSSR interpolation ${%s} is not allowed inside <script> blocks.", variableName)
				} else if blockContext == "style" {
					return fmt.Errorf("GoSSR interpolation ${%s} is not allowed inside <style> blocks.", variableName)
				} else if blockContext == "comment" {
					return fmt.Errorf("GoSSR interpolation ${%s} is not allowed inside HTML comments.", variableName)
				}
			}

			if inTag && currentAttribute != "" {
				lowerAttribute := strings.ToLower(currentAttribute)
				if isExecutableAttribute(lowerAttribute) {
					return fmt.Errorf("GoSSR interpolation ${%s} is not allowed inside executable attribute '%s'.", variableName, currentAttribute)
				}
			}

			scanIndex += 2
			continue
		}

		currentChar := templateHtml[scanIndex]

		if blockContext == "comment" {
			if scanIndex+2 < templateHtmlLength && strings.HasPrefix(templateHtml[scanIndex:], "-->") {
				scanIndex += 3
				blockContext = ""
				continue
			}
		} else if blockContext == "script" {
			if scanIndex+8 < templateHtmlLength && strings.ToLower(templateHtml[scanIndex:scanIndex+9]) == "</script>" {
				scanIndex += 9
				blockContext = ""
				continue
			}
		} else if blockContext == "style" {
			if scanIndex+7 < templateHtmlLength && strings.ToLower(templateHtml[scanIndex:scanIndex+8]) == "</style>" {
				scanIndex += 8
				blockContext = ""
				continue
			}
		} else {
			if currentChar == '<' {
				if scanIndex+3 < templateHtmlLength && strings.HasPrefix(templateHtml[scanIndex:], "<!--") {
					blockContext = "comment"
				} else if scanIndex+6 < templateHtmlLength && strings.HasPrefix(strings.ToLower(templateHtml[scanIndex:]), "<script") {
					nextByte := byte('>')
					if scanIndex+7 < templateHtmlLength {
						nextByte = templateHtml[scanIndex+7]
					}
					if nextByte == ' ' || nextByte == '\t' || nextByte == '\n' || nextByte == '\r' || nextByte == '>' {
						blockContext = "script"
					}
				} else if scanIndex+5 < templateHtmlLength && strings.HasPrefix(strings.ToLower(templateHtml[scanIndex:]), "<style") {
					nextByte := byte('>')
					if scanIndex+6 < templateHtmlLength {
						nextByte = templateHtml[scanIndex+6]
					}
					if nextByte == ' ' || nextByte == '\t' || nextByte == '\n' || nextByte == '\r' || nextByte == '>' {
						blockContext = "style"
					}
				}
				inTag = true
				currentAttribute = ""
				attributeBuffer.Reset()
			} else if inTag {
				if quoteChar != 0 {
					if currentChar == quoteChar {
						quoteChar = 0
						currentAttribute = ""
						attributeBuffer.Reset()
					}
				} else {
					if currentChar == '"' || currentChar == '\'' {
						quoteChar = currentChar
						currentAttribute = strings.TrimSpace(attributeBuffer.String())
					} else if currentChar == '=' {
						currentAttribute = strings.TrimSpace(attributeBuffer.String())
					} else if currentChar == '>' {
						inTag = false
						quoteChar = 0
						currentAttribute = ""
						attributeBuffer.Reset()
					} else if currentChar == ' ' || currentChar == '\t' || currentChar == '\n' || currentChar == '\r' {
						attributeBuffer.Reset()
					} else {
						attributeBuffer.WriteByte(currentChar)
					}
				}
			}
		}

		scanIndex++
	}

	return nil
}

func isWhitespaceCharacter(charByte byte) bool {
	return charByte == ' ' || charByte == '\t' || charByte == '\n' || charByte == '\r'
}

var componentRegistry sync.Map

// RegisterTag registers a custom component tag name with a factory function.
func RegisterTag(tagName string, factory any) {
	componentRegistry.Store(tagName, factory)
}

// Register registers a custom component tag name with a factory function.
func Register(tagName string, factory any) {
	componentRegistry.Store(tagName, factory)
}

func processCustomComponentTags(templateHtml string, currentDepth int) (string, error) {
	if templateHtml == "" {
		return "", nil
	}

	var stringBuilder strings.Builder
	templateHtmlLength := len(templateHtml)
	scanIndex := 0

	for scanIndex < templateHtmlLength {
		openBracket := strings.IndexByte(templateHtml[scanIndex:], '<')
		if openBracket == -1 {
			stringBuilder.WriteString(templateHtml[scanIndex:])
			break
		}

		stringBuilder.WriteString(templateHtml[scanIndex : scanIndex+openBracket])
		scanIndex += openBracket

		// Skip HTML comments <!-- ... -->
		if scanIndex+4 <= templateHtmlLength && strings.HasPrefix(templateHtml[scanIndex:], "<!--") {
			commentEnd := strings.Index(templateHtml[scanIndex+4:], "-->")
			if commentEnd != -1 {
				endPos := scanIndex + 4 + commentEnd + 3
				stringBuilder.WriteString(templateHtml[scanIndex:endPos])
				scanIndex = endPos
				continue
			}
		}

		// Skip <script>...</script>
		if scanIndex+7 <= templateHtmlLength && strings.HasPrefix(strings.ToLower(templateHtml[scanIndex:]), "<script") {
			tagClose := strings.IndexByte(templateHtml[scanIndex:], '>')
			if tagClose != -1 {
				scriptEnd := strings.Index(strings.ToLower(templateHtml[scanIndex+tagClose+1:]), "</script>")
				if scriptEnd != -1 {
					endPos := scanIndex + tagClose + 1 + scriptEnd + 9
					stringBuilder.WriteString(templateHtml[scanIndex:endPos])
					scanIndex = endPos
					continue
				}
			}
		}

		// Skip <style>...</style>
		if scanIndex+6 <= templateHtmlLength && strings.HasPrefix(strings.ToLower(templateHtml[scanIndex:]), "<style") {
			tagClose := strings.IndexByte(templateHtml[scanIndex:], '>')
			if tagClose != -1 {
				styleEnd := strings.Index(strings.ToLower(templateHtml[scanIndex+tagClose+1:]), "</style>")
				if styleEnd != -1 {
					endPos := scanIndex + tagClose + 1 + styleEnd + 8
					stringBuilder.WriteString(templateHtml[scanIndex:endPos])
					scanIndex = endPos
					continue
				}
			}
		}

		tagStart := scanIndex + 1
		if tagStart < templateHtmlLength && isUppercaseLetter(templateHtml[tagStart]) {
			tagNameEnd := tagStart
			for tagNameEnd < templateHtmlLength && isLetterOrDigit(templateHtml[tagNameEnd]) {
				tagNameEnd++
			}

			tagName := templateHtml[tagStart:tagNameEnd]
			if tagNameEnd < templateHtmlLength {
				delimiter := templateHtml[tagNameEnd]
				if isWhitespaceCharacter(delimiter) || delimiter == '/' || delimiter == '>' {
					if factory, registered := componentRegistry.Load(tagName); registered {
						tagEnd := -1
						var inQuote byte = 0
						for tagScanIndex := tagNameEnd; tagScanIndex < templateHtmlLength; tagScanIndex++ {
							currentChar := templateHtml[tagScanIndex]
							if inQuote != 0 {
								if currentChar == inQuote {
									inQuote = 0
								}
							} else {
								if currentChar == '"' || currentChar == '\'' {
									inQuote = currentChar
								} else if currentChar == '>' {
									tagEnd = tagScanIndex
									break
								}
							}
						}

						if tagEnd != -1 {
							rawTagContent := strings.TrimSpace(templateHtml[tagNameEnd:tagEnd])
							isSelfClosing := strings.HasSuffix(rawTagContent, "/")
							attributeString := rawTagContent
							if isSelfClosing {
								attributeString = strings.TrimSpace(rawTagContent[:len(rawTagContent)-1])
							}

							attributes := parseCustomTagAttributes(attributeString)

							nextIndex := tagEnd + 1
							if !isSelfClosing {
								matchingClose := findMatchingClosingComponentTag(templateHtml, tagEnd+1, tagName)
								if matchingClose == -1 {
									return "", fmt.Errorf("Unclosed GoSSR component tag <%s>. Expected self-closing tag <%s ... /> or matching closing tag </%s>", tagName, tagName, tagName)
								}
								bodyContent := templateHtml[tagEnd+1 : matchingClose]
								attributes["children"] = bodyContent
								attributes["content"] = bodyContent
								nextIndex = matchingClose + len("</"+tagName+">")
							}

							renderedChild, err := instantiateAndRenderComponentFactory(factory, attributes, currentDepth+1)
							if err != nil {
								return "", fmt.Errorf("Error rendering component tag <%s>: %w", tagName, err)
							}

							stringBuilder.WriteString(renderedChild)
							scanIndex = nextIndex
							continue
						}
					}
				}
			}
		}

		stringBuilder.WriteByte('<')
		scanIndex++
	}

	return stringBuilder.String(), nil
}

func isUppercaseLetter(charByte byte) bool {
	return charByte >= 'A' && charByte <= 'Z'
}

func isLetterOrDigit(charByte byte) bool {
	return (charByte >= 'a' && charByte <= 'z') || (charByte >= 'A' && charByte <= 'Z') || (charByte >= '0' && charByte <= '9') || charByte == '_'
}

func parseCustomTagAttributes(attributeString string) map[string]string {
	attributes := make(map[string]string)
	if strings.TrimSpace(attributeString) == "" {
		return attributes
	}

	attributeStringLength := len(attributeString)
	scanIndex := 0
	for scanIndex < attributeStringLength {
		for scanIndex < attributeStringLength && isWhitespaceCharacter(attributeString[scanIndex]) {
			scanIndex++
		}
		if scanIndex >= attributeStringLength {
			break
		}

		keyStart := scanIndex
		for scanIndex < attributeStringLength && attributeString[scanIndex] != '=' && attributeString[scanIndex] != '/' && attributeString[scanIndex] != '>' && !isWhitespaceCharacter(attributeString[scanIndex]) {
			scanIndex++
		}
		key := strings.TrimSpace(attributeString[keyStart:scanIndex])
		if key == "" {
			scanIndex++
			continue
		}

		for scanIndex < attributeStringLength && isWhitespaceCharacter(attributeString[scanIndex]) {
			scanIndex++
		}

		value := ""
		if scanIndex < attributeStringLength && attributeString[scanIndex] == '=' {
			scanIndex++
			for scanIndex < attributeStringLength && isWhitespaceCharacter(attributeString[scanIndex]) {
				scanIndex++
			}
			if scanIndex < attributeStringLength {
				if attributeString[scanIndex] == '"' || attributeString[scanIndex] == '\'' {
					quote := attributeString[scanIndex]
					scanIndex++
					valueStart := scanIndex
					for scanIndex < attributeStringLength && attributeString[scanIndex] != quote {
						scanIndex++
					}
					value = attributeString[valueStart:scanIndex]
					if scanIndex < attributeStringLength {
						scanIndex++
					}
				} else {
					valueStart := scanIndex
					for scanIndex < attributeStringLength && !isWhitespaceCharacter(attributeString[scanIndex]) && attributeString[scanIndex] != '/' && attributeString[scanIndex] != '>' {
						scanIndex++
					}
					value = attributeString[valueStart:scanIndex]
				}
			}
		} else {
			value = "true"
		}

		attributes[key] = html.UnescapeString(value)
	}
	return attributes
}

func findMatchingClosingComponentTag(templateHtml string, searchFrom int, tagName string) int {
	openPrefix := "<" + tagName
	closeTag := "</" + tagName + ">"
	templateHtmlLength := len(templateHtml)
	depth := 1
	currentPosition := searchFrom

	for currentPosition < templateHtmlLength {
		nextOpen := strings.Index(templateHtml[currentPosition:], openPrefix)
		if nextOpen != -1 {
			nextOpen += currentPosition
		}
		nextClose := strings.Index(templateHtml[currentPosition:], closeTag)
		if nextClose != -1 {
			nextClose += currentPosition
		}

		if nextClose == -1 {
			return -1
		}

		isValidOpen := false
		isSelfClosing := false
		if nextOpen != -1 && nextOpen < nextClose {
			endNameIndex := nextOpen + len(openPrefix)
			if endNameIndex < templateHtmlLength {
				currentChar := templateHtml[endNameIndex]
				if isWhitespaceCharacter(currentChar) || currentChar == '/' || currentChar == '>' {
					isValidOpen = true
					tagCloseIndex := strings.IndexByte(templateHtml[endNameIndex:], '>')
					if tagCloseIndex != -1 {
						rawContent := strings.TrimSpace(templateHtml[endNameIndex : endNameIndex+tagCloseIndex])
						if strings.HasSuffix(rawContent, "/") {
							isSelfClosing = true
						}
					}
				}
			}
		}

		if isValidOpen && nextOpen < nextClose {
			if !isSelfClosing {
				depth++
			}
			currentPosition = nextOpen + len(openPrefix)
		} else {
			depth--
			if depth == 0 {
				return nextClose
			}
			currentPosition = nextClose + len(closeTag)
		}
	}

	return -1
}

func instantiateAndRenderComponentFactory(factory any, attributes map[string]string, currentDepth int) (string, error) {
	if currentDepth > MaxRenderDepth {
		return "", fmt.Errorf("GoSSR component recursion limit exceeded (max depth %d)", MaxRenderDepth)
	}

	factoryValue := reflect.ValueOf(factory)
	if !factoryValue.IsValid() {
		return "", fmt.Errorf("invalid component factory")
	}

	factoryType := factoryValue.Type()
	if factoryType.Kind() != reflect.Func || factoryType.NumIn() != 1 {
		return "", fmt.Errorf("component factory must be a function taking 1 argument")
	}

	paramType := factoryType.In(0)
	var argumentValue reflect.Value

	if paramType.Kind() == reflect.Map && paramType.Key().Kind() == reflect.String && paramType.Elem().Kind() == reflect.String {
		argumentValue = reflect.ValueOf(attributes)
	} else if paramType.Kind() == reflect.Map && paramType.Key().Kind() == reflect.String && paramType.Elem().Kind() == reflect.Interface {
		interfaceMap := make(map[string]any)
		for attributeKey, attributeValue := range attributes {
			interfaceMap[attributeKey] = attributeValue
		}
		argumentValue = reflect.ValueOf(interfaceMap)
	} else if paramType.Kind() == reflect.Struct {
		validFields := make(map[string]bool)
		var validNames []string
		for fieldIndex := 0; fieldIndex < paramType.NumField(); fieldIndex++ {
			fieldName := paramType.Field(fieldIndex).Name
			validFields[strings.ToLower(fieldName)] = true
			validNames = append(validNames, fieldName)
		}

		for attributeKey := range attributes {
			if attributeKey == "children" || attributeKey == "content" {
				continue
			}
			if !validFields[strings.ToLower(attributeKey)] {
				return "", fmt.Errorf("unrecognized attribute '%s' for custom tag component. Valid attributes: %v", attributeKey, validNames)
			}
		}

		structPtr := reflect.New(paramType)
		structVal := structPtr.Elem()

		for attributeKey, attributeValue := range attributes {
			field := structVal.FieldByName(attributeKey)
			if !field.IsValid() {
				for fieldIndex := 0; fieldIndex < paramType.NumField(); fieldIndex++ {
					structField := paramType.Field(fieldIndex)
					if strings.EqualFold(structField.Name, attributeKey) {
						field = structVal.Field(fieldIndex)
						break
					}
				}
			}

			if field.IsValid() && field.CanSet() {
				switch field.Kind() {
				case reflect.String:
					field.SetString(attributeValue)
				case reflect.Bool:
					field.SetBool(attributeValue == "true" || attributeValue == "TRUE" || attributeValue == "1" || attributeValue == "")
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					var integerValue int64
					fmt.Sscanf(attributeValue, "%d", &integerValue)
					field.SetInt(integerValue)
				case reflect.Struct, reflect.Interface:
					rawVal := reflect.ValueOf(Raw(attributeValue))
					safeUrlVal := reflect.ValueOf(URL(attributeValue))

					if field.Type() == reflect.TypeOf(RawHtml{}) {
						field.Set(rawVal)
					} else if field.Type() == reflect.TypeOf(SafeUrl{}) {
						field.Set(safeUrlVal)
					} else if rawVal.Type().AssignableTo(field.Type()) {
						field.Set(rawVal)
					}
				}
			}
		}

		argumentValue = structVal
	} else {
		return "", fmt.Errorf("unsupported factory parameter type %v", paramType)
	}

	results := factoryValue.Call([]reflect.Value{argumentValue})
	if len(results) == 0 {
		return "", fmt.Errorf("component factory returned no values")
	}

	resultInstance := results[0].Interface()
	if rc, isRenderComponent := resultInstance.(renderComponent); isRenderComponent {
		rc.depth = currentDepth
		var stringBuilder strings.Builder
		if err := rc.Render(&stringBuilder); err != nil {
			return "", err
		}
		return stringBuilder.String(), nil
	} else if ssr, isSSR := resultInstance.(SSR); isSSR {
		var stringBuilder strings.Builder
		if err := ssr.Render(&stringBuilder); err != nil {
			return "", err
		}
		return stringBuilder.String(), nil
	} else if str, isStr := resultInstance.(string); isStr {
		return str, nil
	}

	return fmt.Sprintf("%v", resultInstance), nil
}
