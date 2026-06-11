package fm

import (
	"fmt"
	"reflect"
)

// goTypeToSchemaName maps a Go reflect.Type onto the Swift schema type-name
// string the C bindings expect ("string", "integer", "number", "boolean",
// "array<elem>", or a struct's name for nested types). Mirrors
// type_conversion._python_type_to_string in the Python SDK.
//
// Pointer types are treated as their element type; the optionality flag is
// reported separately. The second return is the "is optional" flag, true when
// the type is a pointer.
func goTypeToSchemaName(t reflect.Type) (string, bool, error) {
	isOptional := false
	if t.Kind() == reflect.Pointer {
		isOptional = true
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return "string", isOptional, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer", isOptional, nil
	case reflect.Float32, reflect.Float64:
		return "number", isOptional, nil
	case reflect.Bool:
		return "boolean", isOptional, nil
	case reflect.Slice, reflect.Array:
		elem := t.Elem()
		elemName, _, err := goTypeToSchemaName(elem)
		if err != nil {
			return "", isOptional, err
		}
		return fmt.Sprintf("array<%s>", elemName), isOptional, nil
	case reflect.Struct:
		return t.Name(), isOptional, nil
	default:
		return "", isOptional, fmt.Errorf("unsupported type %s", t.String())
	}
}
