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
