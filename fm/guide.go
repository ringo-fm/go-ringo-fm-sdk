package fm

/*
#include <stdlib.h>
#include "FoundationModels.h"
*/
import "C"

import "unsafe"

// GuideKind identifies the kind of constraint a GenerationGuide represents.
type GuideKind int

const (
	GuideAnyOf GuideKind = iota
	GuideConstant
	GuideCount
	GuideElement
	GuideMaxItems
	GuideMaximum
	GuideMinItems
	GuideMinimum
	GuideRange
	GuideRegex
)

// GenerationGuide constrains how a property is generated. Use the package-
// level helpers (AnyOf, Range, Regex, ...) rather than constructing directly.
type GenerationGuide struct {
	Kind GuideKind
	// AnyOf / Constant
	StringValues []string
	// Count / MaxItems / MinItems
	IntValue int
	// Maximum / Minimum
	FloatValue float64
	// Range
	MinFloat, MaxFloat float64
	// Regex
	Pattern string
	// Element (wraps another guide; the inner guide is applied to array elements)
	Inner *GenerationGuide
}

// Constructors -------------------------------------------------------------

func AnyOf(values ...string) *GenerationGuide {
	return &GenerationGuide{Kind: GuideAnyOf, StringValues: append([]string(nil), values...)}
}

func Constant(v string) *GenerationGuide {
	return &GenerationGuide{Kind: GuideConstant, StringValues: []string{v}}
}

func Count(n int) *GenerationGuide {
	return &GenerationGuide{Kind: GuideCount, IntValue: n}
}

func MaxItems(n int) *GenerationGuide {
	return &GenerationGuide{Kind: GuideMaxItems, IntValue: n}
}

func MinItems(n int) *GenerationGuide {
	return &GenerationGuide{Kind: GuideMinItems, IntValue: n}
}

func Maximum(v float64) *GenerationGuide {
	return &GenerationGuide{Kind: GuideMaximum, FloatValue: v}
}

func Minimum(v float64) *GenerationGuide {
	return &GenerationGuide{Kind: GuideMinimum, FloatValue: v}
}

func Range(min, max float64) *GenerationGuide {
	return &GenerationGuide{Kind: GuideRange, MinFloat: min, MaxFloat: max}
}

func Regex(pattern string) *GenerationGuide {
	return &GenerationGuide{Kind: GuideRegex, Pattern: pattern}
}

// Element wraps another guide so that the constraint applies to each element
// of an array property rather than the array itself.
func Element(inner *GenerationGuide) *GenerationGuide {
	return &GenerationGuide{Kind: GuideElement, Inner: inner}
}

// applyTo invokes the appropriate C call to register this guide on a property.
func (g *GenerationGuide) applyTo(propPtr unsafe.Pointer) {
	kind := g.Kind
	wrapped := false
	target := g
	if kind == GuideElement && g.Inner != nil {
		target = g.Inner
		kind = target.Kind
		wrapped = true
	}
	switch kind {
	case GuideAnyOf:
		applyAnyOf(propPtr, target.StringValues, wrapped)
	case GuideConstant:
		applyAnyOf(propPtr, target.StringValues, wrapped)
	case GuideCount:
		C.FMGenerationSchemaPropertyAddCountGuide(C.FMGenerationSchemaPropertyRef(propPtr), C.int(target.IntValue), C.bool(wrapped))
	case GuideMaxItems:
		C.FMGenerationSchemaPropertyAddMaxItemsGuide(C.FMGenerationSchemaPropertyRef(propPtr), C.int(target.IntValue))
	case GuideMinItems:
		C.FMGenerationSchemaPropertyAddMinItemsGuide(C.FMGenerationSchemaPropertyRef(propPtr), C.int(target.IntValue))
	case GuideMaximum:
		C.FMGenerationSchemaPropertyAddMaximumGuide(C.FMGenerationSchemaPropertyRef(propPtr), C.double(target.FloatValue), C.bool(wrapped))
	case GuideMinimum:
		C.FMGenerationSchemaPropertyAddMinimumGuide(C.FMGenerationSchemaPropertyRef(propPtr), C.double(target.FloatValue), C.bool(wrapped))
	case GuideRange:
		C.FMGenerationSchemaPropertyAddRangeGuide(C.FMGenerationSchemaPropertyRef(propPtr), C.double(target.MinFloat), C.double(target.MaxFloat), C.bool(wrapped))
	case GuideRegex:
		cstr := C.CString(target.Pattern)
		defer C.free(unsafe.Pointer(cstr))
		C.FMGenerationSchemaPropertyAddRegex(C.FMGenerationSchemaPropertyRef(propPtr), cstr, C.bool(wrapped))
	}
}

func applyAnyOf(propPtr unsafe.Pointer, values []string, wrapped bool) {
	if len(values) == 0 {
		return
	}
	cStrs := make([]*C.char, len(values))
	for i, v := range values {
		cStrs[i] = C.CString(v)
	}
	defer func() {
		for _, c := range cStrs {
			C.free(unsafe.Pointer(c))
		}
	}()
	// The C signature takes `const char **`, so pass the address of the slice's
	// underlying array.
	C.FMGenerationSchemaPropertyAddAnyOfGuide(
		C.FMGenerationSchemaPropertyRef(propPtr),
		(**C.char)(unsafe.Pointer(&cStrs[0])),
		C.int(len(values)),
		C.bool(wrapped),
	)
}
