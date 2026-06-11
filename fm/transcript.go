package fm

/*
#include <stdlib.h>
#include "FoundationModels.h"
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"runtime"
	"unsafe"
)

// Transcript exposes the full conversation history of a Session.
//
// A Transcript loaded via TranscriptFromJSON owns its own underlying session
// pointer; the one returned by Session.Transcript() shares the session's
// pointer (no independent ownership). Both flavors marshal back to JSON via
// MarshalJSON.
type Transcript struct {
	session *Session       // non-nil when this Transcript is tied to a live Session
	ownPtr  unsafe.Pointer // non-nil when we own the underlying pointer (loaded from JSON)
}

// MarshalJSON returns the transcript's JSON representation, suitable for
// persistence and later reloading via TranscriptFromJSON.
func (t *Transcript) MarshalJSON() ([]byte, error) {
	ptr := t.sessionPtr()
	if ptr == nil {
		return nil, fmt.Errorf("transcript: nil session pointer")
	}
	var code C.int
	var desc *C.char
	jstr := C.FMLanguageModelSessionGetTranscriptJSONString(C.FMLanguageModelSessionRef(ptr), &code, &desc)
	if jstr == nil {
		return nil, errorFromStatus(GenerationErrorCode(code), goStringAndFree(desc))
	}
	out := C.GoString(jstr)
	C.FMFreeString(jstr)
	return []byte(out), nil
}

// AsMap parses the transcript JSON into a generic map (matches the Python
// transcript.to_dict() shape).
func (t *Transcript) AsMap() (map[string]any, error) {
	b, err := t.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// EntryCount returns the number of entries in the transcript without parsing
// the full transcript JSON. Returns 0 on error or when the transcript is empty.
func (t *Transcript) EntryCount() int {
	ptr := t.sessionPtr()
	if ptr == nil {
		return 0
	}
	return int(C.FMLanguageModelSessionGetTranscriptEntryCount(C.FMLanguageModelSessionRef(ptr)))
}

// Close releases the underlying C handle, if this Transcript owns one. Safe
// to call multiple times; no-op on Transcripts bound to a live Session.
func (t *Transcript) Close() {
	if t.ownPtr != nil {
		C.FMRelease(t.ownPtr)
		t.ownPtr = nil
		runtime.SetFinalizer(t, nil)
	}
}

func (t *Transcript) sessionPtr() unsafe.Pointer {
	if t.session != nil {
		return t.session.ptr
	}
	return t.ownPtr
}

// TranscriptFromJSON loads a transcript from its JSON form. The returned
// Transcript owns its underlying handle and must be passed to
// Session.FromTranscript or have Close called on it when done.
func TranscriptFromJSON(data []byte) (*Transcript, error) {
	cstr := C.CString(string(data))
	defer C.free(unsafe.Pointer(cstr))
	var code C.int
	var desc *C.char
	ptr := C.FMTranscriptCreateFromJSONString(cstr, &code, &desc)
	if ptr == nil {
		return nil, errorFromStatus(GenerationErrorCode(code), goStringAndFree(desc))
	}
	t := &Transcript{ownPtr: unsafe.Pointer(ptr)}
	runtime.SetFinalizer(t, (*Transcript).Close)
	return t, nil
}

// SessionFromTranscript resumes a session from a Transcript. The returned
// Session takes ownership of the transcript's underlying handle (the
// transcript becomes a view onto the new session).
func SessionFromTranscript(transcript *Transcript, opts ...SessionOption) (*Session, error) {
	if transcript == nil {
		return nil, fmt.Errorf("session: transcript is nil")
	}
	ptr := transcript.sessionPtr()
	if ptr == nil {
		return nil, fmt.Errorf("session: transcript has no underlying pointer")
	}

	var o sessionOptions
	for _, opt := range opts {
		opt(&o)
	}
	var modelPtr C.FMSystemLanguageModelRef
	if o.model != nil {
		modelPtr = C.FMSystemLanguageModelRef(o.model.ptr)
	}
	toolPtrs, toolCount := toolRefArray(o.tools)

	newPtr := C.FMLanguageModelSessionCreateFromTranscript(
		C.FMLanguageModelSessionRef(ptr),
		modelPtr,
		toolPtrs,
		C.int(toolCount),
	)
	if newPtr == nil {
		return nil, fmt.Errorf("session: failed to create from transcript")
	}

	// The new session now owns its own retained pointer. If the transcript
	// previously owned a pointer, release it; from now on the transcript
	// references the new session's pointer.
	if transcript.ownPtr != nil {
		C.FMRelease(transcript.ownPtr)
		transcript.ownPtr = nil
		runtime.SetFinalizer(transcript, nil)
	}

	s := &Session{
		ptr:   unsafe.Pointer(newPtr),
		tools: append([]*BridgedTool(nil), o.tools...),
	}
	transcript.session = s
	s.transcript = transcript
	runtime.SetFinalizer(s, (*Session).release)
	return s, nil
}
