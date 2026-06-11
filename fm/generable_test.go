package fm

import "testing"

func TestGeneratedContentFromJSON(t *testing.T) {
	c, err := GeneratedContentFromJSON(`{"name":"Alice","score":99}`)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	j, err := c.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if j == "" {
		t.Fatal("JSON() returned empty string")
	}

	m, err := c.AsMap()
	if err != nil {
		t.Fatal(err)
	}
	if m["name"] != "Alice" {
		t.Fatalf("name = %v, want Alice", m["name"])
	}
}

func TestGeneratedContentValueAsBool(t *testing.T) {
	c, err := GeneratedContentFromJSON(`{"active":true,"disabled":false,"score":42}`)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	v, ok := c.ValueAsBool("active")
	if !ok {
		t.Error("ValueAsBool(active) ok = false, want true")
	}
	if !v {
		t.Error("ValueAsBool(active) = false, want true")
	}

	v2, ok2 := c.ValueAsBool("disabled")
	if !ok2 {
		t.Error("ValueAsBool(disabled) ok = false, want true")
	}
	if v2 {
		t.Error("ValueAsBool(disabled) = true, want false")
	}

	// Numeric property must fail.
	_, ok3 := c.ValueAsBool("score")
	if ok3 {
		t.Error("ValueAsBool(score) ok = true, want false for numeric property")
	}

	// Missing property must fail.
	_, ok4 := c.ValueAsBool("missing")
	if ok4 {
		t.Error("ValueAsBool(missing) ok = true, want false")
	}
}

func TestGeneratedContentValueAsBoolAfterClose(t *testing.T) {
	c, err := GeneratedContentFromJSON(`{"x":true}`)
	if err != nil {
		t.Fatal(err)
	}
	c.Close()

	_, ok := c.ValueAsBool("x")
	if ok {
		t.Error("ValueAsBool on closed content = true, want false")
	}
}

func TestGeneratedContentValueAsFloat64(t *testing.T) {
	c, err := GeneratedContentFromJSON(`{"price":3.14,"count":42,"label":"hello"}`)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// Float property.
	v, ok := c.ValueAsFloat64("price")
	if !ok {
		t.Error("ValueAsFloat64(price) ok = false, want true")
	}
	if diff := v - 3.14; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("ValueAsFloat64(price) = %v, want 3.14", v)
	}

	// Integer property coerced to float64.
	v2, ok2 := c.ValueAsFloat64("count")
	if !ok2 {
		t.Error("ValueAsFloat64(count) ok = false, want true")
	}
	if v2 != 42.0 {
		t.Errorf("ValueAsFloat64(count) = %v, want 42.0", v2)
	}

	// String property must fail.
	_, ok3 := c.ValueAsFloat64("label")
	if ok3 {
		t.Error("ValueAsFloat64(label) ok = true, want false")
	}

	// Missing property must fail.
	_, ok4 := c.ValueAsFloat64("missing")
	if ok4 {
		t.Error("ValueAsFloat64(missing) ok = true, want false")
	}
}

func TestGeneratedContentValueAsFloat64AfterClose(t *testing.T) {
	c, err := GeneratedContentFromJSON(`{"x":1}`)
	if err != nil {
		t.Fatal(err)
	}
	c.Close()

	_, ok := c.ValueAsFloat64("x")
	if ok {
		t.Error("ValueAsFloat64 on closed content = true, want false")
	}
}

func TestGeneratedContentPropertyNames(t *testing.T) {
	c, err := GeneratedContentFromJSON(`{"score":99,"name":"Alice","active":true}`)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	names, err := c.PropertyNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 3 {
		t.Fatalf("PropertyNames() = %v (len %d), want 3 names", names, len(names))
	}
	want := map[string]bool{"active": true, "name": true, "score": true}
	for _, n := range names {
		if !want[n] {
			t.Errorf("unexpected property name %q", n)
		}
	}
}

func TestGeneratedContentPropertyNamesAfterClose(t *testing.T) {
	c, err := GeneratedContentFromJSON(`{"x":1}`)
	if err != nil {
		t.Fatal(err)
	}
	c.Close()

	// PropertyNames on a released GeneratedContent must return an error, not crash.
	_, err = c.PropertyNames()
	if err == nil {
		t.Error("PropertyNames() on closed content returned nil error, want error")
	}
}

func TestGeneratedContentHasProperty(t *testing.T) {
	c, err := GeneratedContentFromJSON(`{"greeting":"hello","count":42}`)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if !c.HasProperty("greeting") {
		t.Error("HasProperty(greeting) = false, want true")
	}
	if !c.HasProperty("count") {
		t.Error("HasProperty(count) = false, want true")
	}
	if c.HasProperty("nonexistent") {
		t.Error("HasProperty(nonexistent) = true, want false")
	}
	if c.HasProperty("") {
		t.Error("HasProperty(\"\") = true, want false")
	}
}

func TestGeneratedContentHasPropertyAfterClose(t *testing.T) {
	c, err := GeneratedContentFromJSON(`{"x":1}`)
	if err != nil {
		t.Fatal(err)
	}
	c.Close()

	// HasProperty on a released GeneratedContent must not crash.
	if c.HasProperty("x") {
		t.Error("HasProperty on closed content = true, want false")
	}
}
