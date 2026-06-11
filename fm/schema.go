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
	"unsafe"
)

// Property describes a single field of a GenerationSchema.
type Property struct {
	Name        string
	Description string
	TypeName    string // Swift schema type name, e.g. "string", "array<integer>"
	Optional    bool
	Guides      []*GenerationGuide
}

// GenerationSchema describes the structure of guided generation output.
type GenerationSchema struct {
	ptr        unsafe.Pointer // FMGenerationSchemaRef
	TypeName   string
	Properties []Property
}

// NewGenerationSchema constructs a schema with the given type name and
// description. Properties are added via AddProperty before passing to the
// session. References to other schemas are added via AddReference.
func NewGenerationSchema(typeName, description string, properties []Property, references []*GenerationSchema) (*GenerationSchema, error) {
	cname := C.CString(typeName)
	defer C.free(unsafe.Pointer(cname))
	var cdesc *C.char
	if description != "" {
		cdesc = C.CString(description)
		defer C.free(unsafe.Pointer(cdesc))
	}
	ptr := C.FMGenerationSchemaCreate(cname, cdesc)
	s := &GenerationSchema{ptr: unsafe.Pointer(ptr), TypeName: typeName, Properties: properties}
	for _, ref := range references {
		C.FMGenerationSchemaAddReferenceSchema(C.FMGenerationSchemaRef(s.ptr), C.FMGenerationSchemaRef(ref.ptr))
	}
	for _, p := range properties {
		if err := s.addPropertyC(p); err != nil {
			s.release()
			return nil, err
		}
	}
	runtime.SetFinalizer(s, (*GenerationSchema).release)
	return s, nil
}

func (s *GenerationSchema) addPropertyC(p Property) error {
	cname := C.CString(p.Name)
	defer C.free(unsafe.Pointer(cname))
	var cdesc *C.char
	if p.Description != "" {
		cdesc = C.CString(p.Description)
		defer C.free(unsafe.Pointer(cdesc))
	}
	ctype := C.CString(p.TypeName)
	defer C.free(unsafe.Pointer(ctype))
	propPtr := C.FMGenerationSchemaPropertyCreate(cname, cdesc, ctype, C.bool(p.Optional))
	if propPtr == nil {
		return fmt.Errorf("schema: failed to create property %q", p.Name)
	}
	for _, g := range p.Guides {
		g.applyTo(unsafe.Pointer(propPtr))
	}
	C.FMGenerationSchemaAddProperty(C.FMGenerationSchemaRef(s.ptr), propPtr)
	C.FMRelease(unsafe.Pointer(propPtr))
	return nil
}

// JSON returns the schema's JSON Schema representation.
func (s *GenerationSchema) JSON() ([]byte, error) {
	var code C.int
	var desc *C.char
	jstr := C.FMGenerationSchemaGetJSONString(C.FMGenerationSchemaRef(s.ptr), &code, &desc)
	if jstr == nil {
		return nil, errorFromStatus(GenerationErrorCode(code), goStringAndFree(desc))
	}
	out := C.GoString(jstr)
	C.FMFreeString(jstr)
	return []byte(out), nil
}

// AsMap parses the schema JSON into a generic map.
func (s *GenerationSchema) AsMap() (map[string]any, error) {
	b, err := s.JSON()
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Close releases the underlying C resources.
func (s *GenerationSchema) Close() {
	s.release()
	runtime.SetFinalizer(s, nil)
}

func (s *GenerationSchema) release() {
	if s == nil || s.ptr == nil {
		return
	}
	C.FMRelease(s.ptr)
	s.ptr = nil
}

// SchemaFromGoType walks a Go type and builds a GenerationSchema. The type
// must be a struct (or pointer to struct). Field tags use the `fm` namespace:
//
//	type Cat struct {
//	    Name string  `fm:"name,description=Cat's name"`
//	    Age  int     `fm:"age,description=Age in years,range=0:20"`
//	    Food string  `fm:"food,anyOf=fish|chicken|tuna"`
//	    Bio  *string `fm:"bio"` // pointer → optional
//	}
//
// Supported tag keys: description, anyOf (|-separated), constant,
// count, minItems, maxItems, minimum, maximum, range (min:max), regex.
//
// Nested struct fields produce a referenced schema added via
// FMGenerationSchemaAddReferenceSchema. Self-references are detected and
// skipped (matches Python's resolve_referenced_generables).
func SchemaFromGoType(t reflect.Type) (*GenerationSchema, error) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("SchemaFromGoType: expected struct, got %s", t.Kind())
	}
	return buildSchema(t, nil)
}

// SchemaFor is a generic wrapper around SchemaFromGoType.
func SchemaFor[T any]() (*GenerationSchema, error) {
	var zero T
	return SchemaFromGoType(reflect.TypeOf(zero))
}

func buildSchema(t reflect.Type, seen map[reflect.Type]bool) (*GenerationSchema, error) {
	if seen == nil {
		seen = map[reflect.Type]bool{}
	}
	seen[t] = true

	props := make([]Property, 0, t.NumField())
	var references []*GenerationSchema

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := parseFMTag(f)
		name := tag.name
		if name == "" {
			name = f.Name
		}

		typeName, optional, err := goTypeToSchemaName(f.Type)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", f.Name, err)
		}

		// Recurse into nested struct types (including struct slices) for
		// references. Self-references are skipped.
		nestedTypes := collectStructTypes(f.Type)
		for _, nt := range nestedTypes {
			if nt == t || seen[nt] {
				continue
			}
			sub, err := buildSchema(nt, seen)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w", f.Name, err)
			}
			references = append(references, sub)
		}

		props = append(props, Property{
			Name:        name,
			Description: tag.description,
			TypeName:    typeName,
			Optional:    optional,
			Guides:      tag.guides,
		})
	}

	return NewGenerationSchema(t.Name(), "", props, references)
}

func collectStructTypes(t reflect.Type) []reflect.Type {
	switch t.Kind() {
	case reflect.Pointer:
		return collectStructTypes(t.Elem())
	case reflect.Slice, reflect.Array:
		return collectStructTypes(t.Elem())
	case reflect.Struct:
		return []reflect.Type{t}
	default:
		return nil
	}
}
