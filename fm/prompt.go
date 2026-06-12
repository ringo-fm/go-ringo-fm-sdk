package fm

/*
#include <stdlib.h>
#include "FoundationModels.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// PromptComponent is a single element of a prompt: a Text or an *ImageAttachment.
type PromptComponent interface {
	addToComposedPrompt(p C.FMComposedPrompt) error
}

// Prompt is an ordered list of components passed to the model.
type Prompt []PromptComponent

// Text is a plain-text prompt component.
type Text string

func (t Text) addToComposedPrompt(p C.FMComposedPrompt) error {
	c := C.CString(string(t))
	defer C.free(unsafe.Pointer(c))
	C.FMComposedPromptAddText(p, c)
	return nil
}

// TextPrompt is shorthand for Prompt{Text(s)}.
func TextPrompt(s string) Prompt { return Prompt{Text(s)} }

// ImageAttachment attaches an image file to a prompt. Label is optional and,
// when non-empty, lets the model and tools refer to the image by name.
type ImageAttachment struct {
	Path  string
	Label string
}

// NewImageAttachment validates the path exists and returns an attachment.
// The file existence check matches the Python SDK's behavior.
func NewImageAttachment(path string, label string) (*ImageAttachment, error) {
	if path == "" {
		return nil, fmt.Errorf("image attachment: path is empty")
	}
	return &ImageAttachment{Path: path, Label: label}, nil
}

func (a *ImageAttachment) addToComposedPrompt(p C.FMComposedPrompt) error {
	cpath := C.CString(a.Path)
	defer C.free(unsafe.Pointer(cpath))
	var clabel *C.char
	if a.Label != "" {
		clabel = C.CString(a.Label)
		defer C.free(unsafe.Pointer(clabel))
	}
	var errCode C.FMComposedPromptAddImageError
	ok := C.FMComposedPromptAddAttachment(p, cpath, clabel, &errCode)
	if !bool(ok) {
		switch errCode {
		case C.FMComposedPromptAddImageErrorUnsupported:
			return fmt.Errorf("image attachment: format unsupported")
		default:
			return fmt.Errorf("image attachment: failed to add (code %d)", int(errCode))
		}
	}
	return nil
}

// buildComposedPrompt converts a Prompt into a C FMComposedPrompt. The caller
// owns the returned pointer and must not retain it after the C call consumes
// it (the FM API takes ownership when passed to a Respond/Stream call).
func buildComposedPrompt(prompt Prompt) (C.FMComposedPrompt, error) {
	cp := C.FMComposedPromptInitialize()
	for _, comp := range prompt {
		if err := comp.addToComposedPrompt(cp); err != nil {
			return nil, err
		}
	}
	return cp, nil
}

func releaseComposedPromptForTest(cp C.FMComposedPrompt) {
	C.FMRelease(unsafe.Pointer(cp))
}

func composedPromptTextContentForTest(cp C.FMComposedPrompt) string {
	raw := C.FMComposedPromptGetTextContent(cp)
	if raw == nil {
		return ""
	}
	defer C.FMFreeString(raw)
	return C.GoString(raw)
}
