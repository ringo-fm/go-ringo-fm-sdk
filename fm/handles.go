package fm

/*
#include <stdlib.h>
#include "FoundationModels.h"
*/
import "C"

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// The cgo runtime forbids passing Go pointers into C, so we allocate a
// monotonically-increasing uintptr token per registration, store the Go
// object in a sync.Map keyed by that token, and pass the token as the void*
// userInfo parameter. Exported callbacks look the token up and dispatch.

var (
	handleCounter uint64
	handleMap     sync.Map // map[uintptr]any
)

// registerHandle stores obj and returns an opaque token suitable for passing
// to C as a void*. Pair every call with releaseHandle.
func registerHandle(obj any) unsafe.Pointer {
	id := atomic.AddUint64(&handleCounter, 1)
	handleMap.Store(uintptr(id), obj)
	return unsafe.Pointer(uintptr(id))
}

func lookupHandle(p unsafe.Pointer) (any, bool) {
	if p == nil {
		return nil, false
	}
	return handleMap.Load(uintptr(p))
}

func releaseHandle(p unsafe.Pointer) {
	if p == nil {
		return
	}
	handleMap.Delete(uintptr(p))
}

// responseHandle is the Go-side state for a single text/structured response.
// One of textResult or structuredResult is filled in by the callback before
// done is closed.
type responseHandle struct {
	textResult       string
	structuredResult unsafe.Pointer // FMGeneratedContentRef, ownership transferred
	status           GenerationErrorCode
	done             chan struct{}
}

// streamHandle bridges the C iteration callback to a Go channel. The C side
// invokes the callback repeatedly with snapshot bytes, then once with
// length=0 to signal completion. Errors arrive via a non-success status.
type streamHandle struct {
	snapshots chan string
	errCh     chan error
	closed    atomic.Bool
	done      chan struct{}
}

func (s *streamHandle) finish(err error) {
	if !s.closed.CompareAndSwap(false, true) {
		return
	}
	if err != nil {
		s.errCh <- err
	}
	close(s.snapshots)
	close(s.errCh)
	close(s.done)
}

//export goSessionResponseCallback
func goSessionResponseCallback(status C.int, content *C.char, length C.size_t, userInfo unsafe.Pointer) {
	obj, ok := lookupHandle(userInfo)
	if !ok {
		return
	}
	switch h := obj.(type) {
	case *responseHandle:
		h.status = GenerationErrorCode(status)
		if content != nil && length > 0 {
			h.textResult = C.GoStringN(content, C.int(length))
		}
		close(h.done)
	case *streamHandle:
		code := GenerationErrorCode(status)
		if code != codeSuccess {
			h.finish(errorFromStatus(code, ""))
			return
		}
		if content == nil || length == 0 {
			h.finish(nil)
			return
		}
		// Streaming yields full snapshots, not deltas.
		h.snapshots <- C.GoStringN(content, C.int(length))
	}
}

//export goSessionStructuredCallback
func goSessionStructuredCallback(status C.int, content C.FMGeneratedContentRef, userInfo unsafe.Pointer) {
	obj, ok := lookupHandle(userInfo)
	if !ok {
		// Caller is gone; release the retained content to avoid a leak.
		if content != nil {
			C.FMRelease(unsafe.Pointer(content))
		}
		return
	}
	h, ok := obj.(*responseHandle)
	if !ok {
		if content != nil {
			C.FMRelease(unsafe.Pointer(content))
		}
		return
	}
	h.status = GenerationErrorCode(status)
	// Ownership of content transfers to the Go side per Swift passRetained
	// semantics; GeneratedContent's finalizer / explicit Close will FMRelease.
	h.structuredResult = unsafe.Pointer(content)
	close(h.done)
}

// --- Tool slot dispatch ------------------------------------------------------
// The C tool-callback signature has no userInfo, so the C side keeps a fixed
// pool of distinct trampolines (see bridge.c) each carrying its own slot
// index. The Go ledger of BridgedTool-per-slot lives in tool.go; this file
// only provides the shared mutex and the //export entry point.

var toolSlotMu sync.Mutex

//export goToolCallbackSlot
func goToolCallbackSlot(slot C.int, content C.FMGeneratedContentRef, callID C.uint) {
	dispatchToolCall(int(slot), unsafe.Pointer(content), uint32(callID))
}
