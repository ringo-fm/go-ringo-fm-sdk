package fm

/*
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

// Session is a single conversation with a foundation model. Sessions are not
// safe for concurrent Respond/Stream calls; a session-level mutex serializes
// access. Use multiple sessions if you need concurrent requests.
type Session struct {
	ptr        unsafe.Pointer // FMLanguageModelSessionRef
	transcript *Transcript    // shares ptr; updated when ptr changes
	tools      []*BridgedTool // keep tool wrappers alive for the session's lifetime
	mu         sync.Mutex
}

// SessionOption configures NewSession.
type SessionOption func(*sessionOptions)

type sessionOptions struct {
	instructions string
	model        *SystemLanguageModel
	tools        []*BridgedTool
}

func WithInstructions(s string) SessionOption {
	return func(o *sessionOptions) { o.instructions = s }
}

func WithModel(m *SystemLanguageModel) SessionOption {
	return func(o *sessionOptions) { o.model = m }
}

func WithTools(tools ...*BridgedTool) SessionOption {
	return func(o *sessionOptions) { o.tools = append(o.tools, tools...) }
}

// NewSession creates a new language model session.
func NewSession(opts ...SessionOption) (*Session, error) {
	var o sessionOptions
	for _, opt := range opts {
		opt(&o)
	}

	var modelPtr C.FMSystemLanguageModelRef
	if o.model != nil {
		modelPtr = C.FMSystemLanguageModelRef(o.model.ptr)
	}

	var cInstr *C.char
	if o.instructions != "" {
		cInstr = C.CString(o.instructions)
		defer C.free(unsafe.Pointer(cInstr))
	}

	toolPtrs, toolCount := toolRefArray(o.tools)
	ptr := C.FMLanguageModelSessionCreateFromSystemLanguageModel(modelPtr, cInstr, toolPtrs, C.int(toolCount))
	if ptr == nil {
		return nil, fmt.Errorf("session: failed to create")
	}

	s := &Session{
		ptr:   unsafe.Pointer(ptr),
		tools: append([]*BridgedTool(nil), o.tools...),
	}
	s.transcript = &Transcript{session: s}
	runtime.SetFinalizer(s, (*Session).release)
	return s, nil
}

func toolRefArray(tools []*BridgedTool) (*C.FMBridgedToolRef, int) {
	if len(tools) == 0 {
		return nil, 0
	}
	arr := make([]C.FMBridgedToolRef, len(tools))
	for i, t := range tools {
		arr[i] = C.FMBridgedToolRef(t.ptr)
	}
	return (*C.FMBridgedToolRef)(unsafe.Pointer(&arr[0])), len(tools)
}

// IsResponding reports whether a request is in flight on this session.
func (s *Session) IsResponding() bool {
	return bool(C.FMLanguageModelSessionIsResponding(C.FMLanguageModelSessionRef(s.ptr)))
}

// Prewarm asks the system to pre-load resources for this session so the first
// request has lower latency. promptPrefix, when non-empty, is the prefix the
// next prompt is expected to start with. Prewarm is a fire-and-forget hint and
// is safe to call regardless of model availability.
func (s *Session) Prewarm(promptPrefix string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ptr == nil {
		return
	}
	var cPrefix *C.char
	if promptPrefix != "" {
		cPrefix = C.CString(promptPrefix)
		defer C.free(unsafe.Pointer(cPrefix))
	}
	C.FMLanguageModelSessionPrewarm(C.FMLanguageModelSessionRef(s.ptr), cPrefix)
}

// Transcript returns the transcript view of this session. The transcript
// shares the session's pointer and is updated as the session progresses.
func (s *Session) Transcript() *Transcript { return s.transcript }

// Close releases the underlying C resources.
func (s *Session) Close() {
	s.release()
	runtime.SetFinalizer(s, nil)
}

func (s *Session) release() {
	if s == nil || s.ptr == nil {
		return
	}
	C.FMRelease(s.ptr)
	s.ptr = nil
}

func (s *Session) reset() {
	if s.ptr != nil {
		C.FMLanguageModelSessionReset(C.FMLanguageModelSessionRef(s.ptr))
	}
}

// RespondOption configures a single Respond/Stream call.
type RespondOption func(*respondOptions)

type respondOptions struct {
	options *GenerationOptions
}

func WithGenerationOptions(g GenerationOptions) RespondOption {
	return func(o *respondOptions) { o.options = &g }
}

// Respond sends a prompt and returns the model's text response.
func (s *Session) Respond(ctx context.Context, prompt Prompt, opts ...RespondOption) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, err := buildComposedPrompt(prompt)
	if err != nil {
		return "", err
	}
	optsJSON, err := optionsJSON(opts)
	if err != nil {
		return "", err
	}
	var cOpts *C.char
	if optsJSON != "" {
		cOpts = C.CString(optsJSON)
		defer C.free(unsafe.Pointer(cOpts))
	}

	h := &responseHandle{done: make(chan struct{})}
	token := registerHandle(h)
	defer releaseHandle(token)

	task := C.fm_session_respond(C.FMLanguageModelSessionRef(s.ptr), cp, cOpts, token)
	defer C.FMRelease(unsafe.Pointer(task))

	if err := s.waitFor(ctx, task, h); err != nil {
		return "", err
	}
	if err := errorFromStatus(h.status, ""); err != nil {
		return "", err
	}
	return h.textResult, nil
}

// RespondWithSchema sends a prompt and returns structured content matching
// the provided schema. Caller owns the returned *GeneratedContent.
func (s *Session) RespondWithSchema(ctx context.Context, prompt Prompt, schema *GenerationSchema, opts ...RespondOption) (*GeneratedContent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, err := buildComposedPrompt(prompt)
	if err != nil {
		return nil, err
	}
	optsJSON, err := optionsJSON(opts)
	if err != nil {
		return nil, err
	}
	var cOpts *C.char
	if optsJSON != "" {
		cOpts = C.CString(optsJSON)
		defer C.free(unsafe.Pointer(cOpts))
	}

	h := &responseHandle{done: make(chan struct{})}
	token := registerHandle(h)
	defer releaseHandle(token)

	task := C.fm_session_respond_with_schema(
		C.FMLanguageModelSessionRef(s.ptr),
		cp,
		C.FMGenerationSchemaRef(schema.ptr),
		cOpts,
		token,
	)
	defer C.FMRelease(unsafe.Pointer(task))

	if err := s.waitFor(ctx, task, h); err != nil {
		return nil, err
	}
	if h.status != codeSuccess {
		var dbg string
		if h.structuredResult != nil {
			gc := newGeneratedContent(h.structuredResult)
			if j, jerr := gc.JSON(); jerr == nil {
				dbg = j
			}
			gc.Close()
		}
		return nil, errorFromStatus(h.status, dbg)
	}
	if h.structuredResult == nil {
		return nil, fmt.Errorf("session: structured response missing content")
	}
	return newGeneratedContent(h.structuredResult), nil
}

// RespondWithJSONSchema is the JSON-schema variant of RespondWithSchema.
func (s *Session) RespondWithJSONSchema(ctx context.Context, prompt Prompt, schemaJSON []byte, opts ...RespondOption) (*GeneratedContent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, err := buildComposedPrompt(prompt)
	if err != nil {
		return nil, err
	}
	optsJSON, err := optionsJSON(opts)
	if err != nil {
		return nil, err
	}
	var cOpts *C.char
	if optsJSON != "" {
		cOpts = C.CString(optsJSON)
		defer C.free(unsafe.Pointer(cOpts))
	}
	cSchema := C.CString(string(schemaJSON))
	defer C.free(unsafe.Pointer(cSchema))

	h := &responseHandle{done: make(chan struct{})}
	token := registerHandle(h)
	defer releaseHandle(token)

	task := C.fm_session_respond_with_schema_json(
		C.FMLanguageModelSessionRef(s.ptr),
		cp,
		cSchema,
		cOpts,
		token,
	)
	defer C.FMRelease(unsafe.Pointer(task))

	if err := s.waitFor(ctx, task, h); err != nil {
		return nil, err
	}
	if h.status != codeSuccess {
		return nil, errorFromStatus(h.status, "")
	}
	if h.structuredResult == nil {
		return nil, fmt.Errorf("session: structured response missing content")
	}
	return newGeneratedContent(h.structuredResult), nil
}

// RespondInto sends a prompt and decodes the structured response into out
// (a pointer to a struct). The schema is derived from out's type via
// SchemaFromGoType.
func (s *Session) RespondInto(ctx context.Context, prompt Prompt, out any, opts ...RespondOption) error {
	if out == nil {
		return fmt.Errorf("RespondInto: out is nil")
	}
	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("RespondInto: out must be a non-nil pointer")
	}
	schema, err := SchemaFromGoType(v.Elem().Type())
	if err != nil {
		return err
	}
	defer schema.Close()
	content, err := s.RespondWithSchema(ctx, prompt, schema, opts...)
	if err != nil {
		return err
	}
	defer content.Close()
	return content.Decode(out)
}

func optionsJSON(opts []RespondOption) (string, error) {
	if len(opts) == 0 {
		return "", nil
	}
	var o respondOptions
	for _, opt := range opts {
		opt(&o)
	}
	if o.options == nil {
		return "", nil
	}
	return o.options.toJSON()
}

// waitFor blocks until h.done closes or ctx is cancelled. On cancellation it
// cancels the native task, waits up to 1 second for is_responding to clear,
// then resets the session — mirroring the Python SDK's cleanup sequence.
func (s *Session) waitFor(ctx context.Context, task C.FMTaskRef, h *responseHandle) error {
	if ctx == nil {
		<-h.done
		return nil
	}
	select {
	case <-h.done:
		return nil
	case <-ctx.Done():
		C.FMTaskCancel(task)
		deadline := time.Now().Add(1 * time.Second)
		for time.Now().Before(deadline) && s.IsResponding() {
			time.Sleep(10 * time.Millisecond)
		}
		s.reset()
		// Drain the callback so the handle's goroutine doesn't leak. The C
		// side calls back with cancelled status; the channel will close.
		select {
		case <-h.done:
		case <-time.After(500 * time.Millisecond):
		}
		return ctx.Err()
	}
}
