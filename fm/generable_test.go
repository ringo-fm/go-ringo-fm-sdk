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
