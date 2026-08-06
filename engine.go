package main

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
)

// SSR represents any component capable of writing HTML directly to an output stream.
type SSR interface {
	Render(writer io.Writer) error
	String() string
}

type renderComponent struct {
	templateString string
	scopeArguments []any
}

func (component renderComponent) String() string {
	var stringBuilder strings.Builder
	_ = component.Render(&stringBuilder)
	return stringBuilder.String()
}

func (component renderComponent) Render(writer io.Writer) error {
	outputHtml := component.templateString

	for _, argument := range component.scopeArguments {
		outputHtml = processScopeArgument(outputHtml, argument)
	}

	_, writeError := io.WriteString(writer, outputHtml)
	return writeError
}

func Render(templateString string, scopeArguments ...any) SSR {
	return renderComponent{
		templateString: templateString,
		scopeArguments: scopeArguments,
	}
}

// ssr is an unexported alias for Render to ensure full backwards compatibility
func ssr(templateString string, scopeArguments ...any) SSR {
	return Render(templateString, scopeArguments...)
}

func processScopeArgument(templateString string, argument any) string {
	reflectionValue := reflect.ValueOf(argument)
	if reflectionValue.Kind() == reflect.Pointer {
		reflectionValue = reflectionValue.Elem()
	}

	if reflectionValue.Kind() == reflect.Struct {
		reflectionType := reflectionValue.Type()
		for index := 0; index < reflectionValue.NumField(); index++ {
			structField := reflectionType.Field(index)
			fieldVal := reflectionValue.Field(index)
			if !structField.IsExported() || !fieldVal.CanInterface() {
				continue
			}

			fieldName := structField.Name
			fieldValue := fieldVal.Interface()

			// Process map expressions for slice fields
			templateString = processMapExpressions(templateString, fieldName, fieldValue)

			// Process top-level ternary expressions
			templateString = processTernaryExpressions(templateString, fieldName, fieldValue)

			// Process direct top-level property replacement
			placeholderPattern := fmt.Sprintf("${properties.%s}", fieldName)
			if renderableObject, isRenderable := fieldValue.(SSR); isRenderable {
				templateString = strings.ReplaceAll(templateString, placeholderPattern, renderableObject.String())
			} else {
				templateString = strings.ReplaceAll(templateString, placeholderPattern, fmt.Sprintf("%v", fieldValue))
			}

			// Process nested field replacements (e.g. ${properties.Task.ID}, ${properties.Task.Title})
			fieldReflectionValue := reflect.ValueOf(fieldValue)
			if fieldReflectionValue.Kind() == reflect.Pointer {
				fieldReflectionValue = fieldReflectionValue.Elem()
			}

			if fieldReflectionValue.Kind() == reflect.Struct {
				nestedType := fieldReflectionValue.Type()
				for nestedIndex := 0; nestedIndex < fieldReflectionValue.NumField(); nestedIndex++ {
					nestedStructField := nestedType.Field(nestedIndex)
					nestedFieldVal := fieldReflectionValue.Field(nestedIndex)
					if !nestedStructField.IsExported() || !nestedFieldVal.CanInterface() {
						continue
					}

					nestedFieldName := nestedStructField.Name
					nestedFieldValue := nestedFieldVal.Interface()

					nestedPattern := fmt.Sprintf("${properties.%s.%s}", fieldName, nestedFieldName)
					if renderableObject, isRenderable := nestedFieldValue.(SSR); isRenderable {
						templateString = strings.ReplaceAll(templateString, nestedPattern, renderableObject.String())
					} else {
						templateString = strings.ReplaceAll(templateString, nestedPattern, fmt.Sprintf("%v", nestedFieldValue))
					}

					// Process nested ternary expressions (e.g. ${properties.Task.Completed ? "done" : "pending"})
					templateString = processNestedTernaryExpressions(templateString, fieldName, nestedFieldName, nestedFieldValue)
				}
			}
		}
	}

	return templateString
}

func isTruthy(value any) bool {
	if value == nil {
		return false
	}
	reflectionValue := reflect.ValueOf(value)
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
	case reflect.Pointer, reflect.Interface:
		return !reflectionValue.IsNil()
	default:
		return true
	}
}

func processTernaryExpressions(templateString string, fieldName string, fieldValue any) string {
	quotedTernaryRegex := regexp.MustCompile(fmt.Sprintf(`\$\{properties\.%s\s*\?\s*"([^"]*)"\s*:\s*"([^"]*)"\}`, fieldName))
	templateString = quotedTernaryRegex.ReplaceAllStringFunc(templateString, func(match string) string {
		submatches := quotedTernaryRegex.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			if isTruthy(fieldValue) {
				return submatches[1]
			}
			return submatches[2]
		}
		return match
	})

	unquotedTernaryRegex := regexp.MustCompile(fmt.Sprintf(`\$\{properties\.%s\s*\?\s*([^:\s\}]+)\s*:\s*([^}\s]+)\}`, fieldName))
	templateString = unquotedTernaryRegex.ReplaceAllStringFunc(templateString, func(match string) string {
		submatches := unquotedTernaryRegex.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			if isTruthy(fieldValue) {
				return strings.Trim(submatches[1], `"`)
			}
			return strings.Trim(submatches[2], `"`)
		}
		return match
	})

	return templateString
}

func processNestedTernaryExpressions(templateString string, parentFieldName string, childFieldName string, fieldValue any) string {
	quotedTernaryRegex := regexp.MustCompile(fmt.Sprintf(`\$\{properties\.%s\.%s\s*\?\s*"([^"]*)"\s*:\s*"([^"]*)"\}`, parentFieldName, childFieldName))
	templateString = quotedTernaryRegex.ReplaceAllStringFunc(templateString, func(match string) string {
		submatches := quotedTernaryRegex.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			if isTruthy(fieldValue) {
				return submatches[1]
			}
			return submatches[2]
		}
		return match
	})

	unquotedTernaryRegex := regexp.MustCompile(fmt.Sprintf(`\$\{properties\.%s\.%s\s*\?\s*([^:\s\}]+)\s*:\s*([^}\s]+)\}`, parentFieldName, childFieldName))
	templateString = unquotedTernaryRegex.ReplaceAllStringFunc(templateString, func(match string) string {
		submatches := unquotedTernaryRegex.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			if isTruthy(fieldValue) {
				return strings.Trim(submatches[1], `"`)
			}
			return strings.Trim(submatches[2], `"`)
		}
		return match
	})

	return templateString
}

func processMapExpressions(templateString string, fieldName string, fieldValue any) string {
	mapRegex := regexp.MustCompile(fmt.Sprintf(`\$\{properties\.%s\.map\([a-zA-Z0-9_]+\s*=>\s*(.*?)\)\}`, fieldName))
	matches := mapRegex.FindStringSubmatch(templateString)

	if len(matches) < 2 {
		return templateString
	}

	sliceValue := reflect.ValueOf(fieldValue)
	if sliceValue.Kind() != reflect.Slice {
		return templateString
	}

	var renderedItems strings.Builder
	for index := 0; index < sliceValue.Len(); index++ {
		item := sliceValue.Index(index).Interface()

		if singleTask, isTask := item.(Task); isTask {
			renderedItems.WriteString(TaskItem(TaskItemProperties{Task: singleTask}).String())
		} else if renderableItem, isRenderable := item.(SSR); isRenderable {
			renderedItems.WriteString(renderableItem.String())
		} else if stringerItem, isStringer := item.(fmt.Stringer); isStringer {
			renderedItems.WriteString(stringerItem.String())
		} else {
			renderedItems.WriteString(fmt.Sprintf("%v", item))
		}
	}

	return mapRegex.ReplaceAllString(templateString, renderedItems.String())
}
