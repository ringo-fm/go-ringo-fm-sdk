package fm

/*
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"context"
	"fmt"
	"runtime"
	"unsafe"
)

// Tool is implemented by user-defined tools the model can invoke during
// generation. Each invocation receives a GeneratedContent containing the
// model's chosen arguments (matching ArgumentsSchema) and returns a string
// result handed back to the model.
type Tool interface {
	Name() string
	Description() string
	ArgumentsSchema() *GenerationSchema
	Call(ctx context.Context, args *GeneratedContent) (string, error)
}

// BridgedTool is the C-side handle for a registered Tool. Construct one with
// RegisterTool and pass it to NewSession via WithTools.
type BridgedTool struct {
	ptr    unsafe.Pointer // FMBridgedToolRef
	slot   int
	tool   Tool
	schema *GenerationSchema // kept alive so the C side can reference its props
}

// Slot-indexed registry of BridgedTool, populated by RegisterTool and read by
// the C callback dispatcher. Guarded by toolSlotMu (defined in handles.go).
// Size must match FM_TOOL_SLOTS in bridge.c.
var bridgedTools [32]*BridgedTool

// acquireToolSlot reserves the first free slot. Caller must hold toolSlotMu
// or arrange exclusion via RegisterTool's natural serialization.
func acquireToolSlot() int {
	toolSlotMu.Lock()
	defer toolSlotMu.Unlock()
	for i := range bridgedTools {
		if bridgedTools[i] == nil {
			// Reserve with a sentinel so a concurrent acquire skips this slot.
			bridgedTools[i] = reservedSentinel
			return i
		}
	}
	return -1
}

// reservedSentinel marks a slot as reserved-but-not-yet-populated so a
// concurrent RegisterTool won't pick the same slot. Replaced with the real
// BridgedTool once construction succeeds, or cleared on failure.
var reservedSentinel = &BridgedTool{}

// RegisterTool wires a Tool into the C bridge. The returned BridgedTool must
// stay alive for the lifetime of any session that uses it; Close releases
// the underlying C handle and frees the slot.
func RegisterTool(t Tool) (*BridgedTool, error) {
	slot := acquireToolSlot()
	if slot < 0 {
		return nil, fmt.Errorf("tool: no free slot (limit %d concurrent tools)", len(bridgedTools))
	}

	schema := t.ArgumentsSchema()
	cname := C.CString(t.Name())
	defer C.free(unsafe.Pointer(cname))
	cdesc := C.CString(t.Description())
	defer C.free(unsafe.Pointer(cdesc))

	var errCode C.int
	var errDesc *C.char
	ptr := C.fm_tool_create_at_slot(
		C.int(slot),
		cname,
		cdesc,
		C.FMGenerationSchemaRef(schema.ptr),
		&errCode,
		&errDesc,
	)
	if ptr == nil {
		toolSlotMu.Lock()
		bridgedTools[slot] = nil
		toolSlotMu.Unlock()
		msg := goStringAndFree(errDesc)
		if e := errorFromStatus(GenerationErrorCode(errCode), msg); e != nil {
			return nil, e
		}
		return nil, fmt.Errorf("tool: failed to create %q", t.Name())
	}

	bt := &BridgedTool{ptr: unsafe.Pointer(ptr), slot: slot, tool: t, schema: schema}

	toolSlotMu.Lock()
	bridgedTools[slot] = bt
	toolSlotMu.Unlock()

	runtime.SetFinalizer(bt, (*BridgedTool).release)
	return bt, nil
}

// Close releases the C handle and frees the slot.
func (b *BridgedTool) Close() {
	b.release()
	runtime.SetFinalizer(b, nil)
}

func (b *BridgedTool) release() {
	if b == nil {
		return
	}
	toolSlotMu.Lock()
	ptr := b.ptr
	b.ptr = nil
	if b.slot >= 0 && b.slot < len(bridgedTools) && bridgedTools[b.slot] == b {
		bridgedTools[b.slot] = nil
	}
	toolSlotMu.Unlock()

	if ptr != nil {
		C.FMRelease(ptr)
	}
}

// dispatchToolCall is invoked from goToolCallbackSlot to deliver a tool call
// to the user's Go Tool. Ownership of contentPtr transfers to the
// GeneratedContent here; we release it when the tool's Call returns.
func dispatchToolCall(slot int, contentPtr unsafe.Pointer, callID uint32) {
	toolSlotMu.Lock()
	bt := (*BridgedTool)(nil)
	if slot >= 0 && slot < len(bridgedTools) {
		bt = bridgedTools[slot]
	}
	toolSlotMu.Unlock()

	if bt == nil || bt == reservedSentinel {
		if contentPtr != nil {
			C.FMRelease(contentPtr)
		}
		return
	}
	t := bt.tool

	go func() {
		content := newGeneratedContent(contentPtr)
		defer content.Close()

		ctx := context.Background()
		out, err := t.Call(ctx, content)
		if err != nil {
			out = fmt.Sprintf("Tool error: %s", err.Error())
		}
		cout := C.CString(out)
		defer C.free(unsafe.Pointer(cout))

		toolSlotMu.Lock()
		ptr := bt.ptr
		toolSlotMu.Unlock()
		if ptr == nil {
			return
		}
		C.FMBridgedToolFinishCall(C.FMBridgedToolRef(ptr), C.uint(callID), cout)
	}()
}
