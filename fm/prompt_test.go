package fm

/*
#include <stdlib.h>
#include "FoundationModels.h"
*/
import "C"
import "testing"

func composedPromptTextContent(cp C.FMComposedPrompt) string {
	raw := C.FMComposedPromptGetTextContent(cp)
	if raw == nil {
		return ""
	}
	defer C.FMFreeString(raw)
	return C.GoString(raw)
}

func TestBuildComposedPromptSingleText(t *testing.T) {
	cp, err := buildComposedPrompt(TextPrompt("hello world"))
	if err != nil {
		t.Fatal(err)
	}
	defer C.FMRelease(cp)

	got := composedPromptTextContent(cp)
	if got != "hello world" {
		t.Errorf("text content = %q, want %q", got, "hello world")
	}
}

func TestBuildComposedPromptMultiText(t *testing.T) {
	cp, err := buildComposedPrompt(Prompt{Text("foo"), Text("bar")})
	if err != nil {
		t.Fatal(err)
	}
	defer C.FMRelease(cp)

	got := composedPromptTextContent(cp)
	if got != "foobar" {
		t.Errorf("text content = %q, want %q", got, "foobar")
	}
}

func TestBuildComposedPromptEmpty(t *testing.T) {
	cp, err := buildComposedPrompt(Prompt{})
	if err != nil {
		t.Fatal(err)
	}
	defer C.FMRelease(cp)

	got := composedPromptTextContent(cp)
	if got != "" {
		t.Errorf("text content = %q, want empty string", got)
	}
}
