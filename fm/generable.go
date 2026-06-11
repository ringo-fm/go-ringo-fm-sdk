package fm

/*
#include <stdlib.h>
#include "FoundationModels.h"
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"unsafe"
)

// GeneratedContent is a piece of model output produced by guided generation.
// It wraps a retained C handle and lazily exposes the data as a parsed map.
type GeneratedContent struct {
	ptr     unsafe.Pointer // FMGeneratedContentRef
	parsed  map[string]any
	hasJSON bool
	rawJSON string
}

// newGeneratedContent takes ownership of a +1-retained content pointer.
func newGeneratedContent(ptr unsafe.Pointer) *GeneratedContent {
	c := &GeneratedContent{ptr: ptr}
	runtime.SetFinalizer(c, (*GeneratedContent).release)
	return c
}

// GeneratedContentFromJSON builds a GeneratedContent from a JSON string,
// matching FMGeneratedContentCreateFromJSON.
func GeneratedContentFromJSON(jsonStr string) (*GeneratedContent, error) {
	cstr := C.CString(jsonStr)
	defer C.free(unsafe.Pointer(cstr))
	var code C.int
	var desc *C.char
	ptr := C.FMGeneratedContentCreateFromJSON(cstr, &code, &desc)
	if ptr == nil {
		return nil, errorFromStatus(GenerationErrorCode(code), goStringAndFree(desc))
	}
	c := newGeneratedContent(unsafe.Pointer(ptr))
	c.rawJSON = jsonStr
	c.hasJSON = true
	return c, nil
}

// JSON returns the underlying content as a JSON string.
func (c *GeneratedContent) JSON() (string, error) {
	if c.hasJSON {
		return c.rawJSON, nil
	}
	if c.ptr == nil {
		return "", fmt.Errorf("generated content: released")
	}
	jstr := C.FMGeneratedContentGetJSONString(C.FMGeneratedContentRef(c.ptr))
	if jstr == nil {
		return "", fmt.Errorf("generated content: failed to read JSON")
	}
	c.rawJSON = C.GoString(jstr)
	C.FMFreeString(jstr)
	c.hasJSON = true
	return c.rawJSON, nil
}

// AsMap parses the content as map[string]any (cached after first call).
func (c *GeneratedContent) AsMap() (map[string]any, error) {
	if c.parsed != nil {
		return c.parsed, nil
	}
	j, err := c.JSON()
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(j), &m); err != nil {
		return nil, err
	}
	c.parsed = m
	return m, nil
}

// Value returns the value for a top-level property. nil if missing.
func (c *GeneratedContent) Value(property string) any {
	m, err := c.AsMap()
	if err != nil {
		return nil
	}
	return m[property]
}

// ValueAsFloat64 returns the value of a numeric top-level property as float64.
// The second return value is false when the property is absent, is not numeric,
// or the content has been released.
func (c *GeneratedContent) ValueAsFloat64(property string) (float64, bool) {
	if c.ptr == nil {
		return 0, false
	}
	cname := C.CString(property)
	defer C.free(unsafe.Pointer(cname))
	var out C.double
	var code C.int
	ok := C.FMGeneratedContentGetPropertyValueAsDouble(C.FMGeneratedContentRef(c.ptr), cname, &out, &code)
	if !bool(ok) {
		return 0, false
	}
	return float64(out), true
}

// PropertyNames returns the sorted list of top-level property names present in
// the content. It is useful for dynamic schema handling when the schema is not
// fully known at compile time. Returns nil on error or after Close.
func (c *GeneratedContent) PropertyNames() ([]string, error) {
	if c.ptr == nil {
		return nil, fmt.Errorf("generated content: released")
	}
	raw := C.FMGeneratedContentGetPropertyNames(C.FMGeneratedContentRef(c.ptr))
	if raw == nil {
		return nil, fmt.Errorf("generated content: failed to read property names")
	}
	defer C.FMFreeString(raw)
	var names []string
	if err := json.Unmarshal([]byte(C.GoString(raw)), &names); err != nil {
		return nil, err
	}
	return names, nil
}

// ValueAsInt64 returns the value of an integer top-level property as int64.
// The second return value is false when the property is absent, is not an
// integer (e.g. a float or string), or the content has been released.
func (c *GeneratedContent) ValueAsInt64(property string) (int64, bool) {
	if c.ptr == nil {
		return 0, false
	}
	cname := C.CString(property)
	defer C.free(unsafe.Pointer(cname))
	var out C.int64_t
	var code C.int
	ok := C.FMGeneratedContentGetPropertyValueAsInt(C.FMGeneratedContentRef(c.ptr), cname, &out, &code)
	if !bool(ok) {
		return 0, false
	}
	return int64(out), true
}

// ValueAsBool returns the value of a boolean top-level property.
// The second return value is false when the property is absent, is not boolean,
// or the content has been released.
func (c *GeneratedContent) ValueAsBool(property string) (bool, bool) {
	if c.ptr == nil {
		return false, false
	}
	cname := C.CString(property)
	defer C.free(unsafe.Pointer(cname))
	var out C.bool
	var code C.int
	ok := C.FMGeneratedContentGetPropertyValueAsBool(C.FMGeneratedContentRef(c.ptr), cname, &out, &code)
	if !bool(ok) {
		return false, false
	}
	return bool(out), true
}

// HasProperty reports whether the content has a top-level property with the
// given name. It is cheaper than calling Value and checking for nil because it
// does not distinguish between a missing key and a key whose value is null.
// Use HasProperty to guard GetPropertyValue calls when the schema is not fully
// known in advance.
func (c *GeneratedContent) HasProperty(name string) bool {
	if c.ptr == nil {
		return false
	}
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return bool(C.FMGeneratedContentHasProperty(C.FMGeneratedContentRef(c.ptr), cname))
}

// IsComplete reports whether the model finished producing this content.
func (c *GeneratedContent) IsComplete() bool {
	if c.ptr == nil {
		return false
	}
	return bool(C.FMGeneratedContentIsComplete(C.FMGeneratedContentRef(c.ptr)))
}

// Decode unmarshals the content into out (a pointer to a struct, map, etc).
// Uses encoding/json so standard json tags are respected on out.
func (c *GeneratedContent) Decode(out any) error {
	j, err := c.JSON()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(j), out)
}

// Close releases the underlying C resources.
func (c *GeneratedContent) Close() {
	c.release()
	runtime.SetFinalizer(c, nil)
}

func (c *GeneratedContent) release() {
	if c == nil || c.ptr == nil {
		return
	}
	C.FMRelease(c.ptr)
	c.ptr = nil
}

// --- Struct-tag parsing for SchemaFromGoType ---------------------------------

type fmTag struct {
	name        string
	description string
	guides      []*GenerationGuide
}

// parseFMTag parses an `fm:"..."` struct tag. Grammar:
//
//	fm:"<name>,key=value,key=value,..."
//
// Recognized keys: description, anyOf (|-separated), constant, count,
// minItems, maxItems, minimum, maximum, range (min:max), regex.
//
// If the tag is empty or absent the field's name is used and no guides are
// attached.
func parseFMTag(f reflect.StructField) fmTag {
	raw := f.Tag.Get("fm")
	if raw == "" {
		return fmTag{name: lowerFirst(f.Name)}
	}
	parts := splitTagCSV(raw)
	t := fmTag{}
	for i, p := range parts {
		if i == 0 && !strings.ContainsRune(p, '=') {
			t.name = strings.TrimSpace(p)
			continue
		}
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "description":
			t.description = v
		case "anyOf":
			t.guides = append(t.guides, AnyOf(strings.Split(v, "|")...))
		case "constant":
			t.guides = append(t.guides, Constant(v))
		case "count":
			if n, err := strconv.Atoi(v); err == nil {
				t.guides = append(t.guides, Count(n))
			}
		case "minItems":
			if n, err := strconv.Atoi(v); err == nil {
				t.guides = append(t.guides, MinItems(n))
			}
		case "maxItems":
			if n, err := strconv.Atoi(v); err == nil {
				t.guides = append(t.guides, MaxItems(n))
			}
		case "minimum":
			if f64, err := strconv.ParseFloat(v, 64); err == nil {
				t.guides = append(t.guides, Minimum(f64))
			}
		case "maximum":
			if f64, err := strconv.ParseFloat(v, 64); err == nil {
				t.guides = append(t.guides, Maximum(f64))
			}
		case "range":
			lo, hi, ok := strings.Cut(v, ":")
			if !ok {
				continue
			}
			loF, err1 := strconv.ParseFloat(lo, 64)
			hiF, err2 := strconv.ParseFloat(hi, 64)
			if err1 == nil && err2 == nil {
				t.guides = append(t.guides, Range(loF, hiF))
			}
		case "regex":
			t.guides = append(t.guides, Regex(v))
		}
	}
	if t.name == "" {
		t.name = lowerFirst(f.Name)
	}
	return t
}

// splitTagCSV splits "a,b=c,d=x|y" into ["a", "b=c", "d=x|y"] without breaking
// inside an anyOf | list.
func splitTagCSV(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'A' && r[0] <= 'Z' {
		r[0] += 'a' - 'A'
	}
	return string(r)
}

