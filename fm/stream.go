package fm

/*
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"context"
	"time"
	"unsafe"
)

// StreamResponse sends a prompt and streams response snapshots. Each value on
// the returned channel is a complete text snapshot (not a delta). The
// snapshots channel closes when the model finishes; any error is sent on the
// errs channel before close. Cancel the context to abort.
//
// The caller MUST drain snapshots (or cancel) — otherwise the underlying
// iteration goroutine will block forever waiting for a reader.
func (s *Session) StreamResponse(ctx context.Context, prompt Prompt, opts ...RespondOption) (<-chan string, <-chan error) {
	snapshots := make(chan string, 16)
	errs := make(chan error, 1)

	h := &streamHandle{snapshots: snapshots, errCh: errs, done: make(chan struct{})}

	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		cp, err := buildComposedPrompt(prompt)
		if err != nil {
			h.finish(err)
			return
		}
		optsJSON, err := optionsJSON(opts)
		if err != nil {
			h.finish(err)
			return
		}
		var cOpts *C.char
		if optsJSON != "" {
			cOpts = C.CString(optsJSON)
			defer C.free(unsafe.Pointer(cOpts))
		}

		streamPtr := C.FMLanguageModelSessionStreamResponse(
			C.FMLanguageModelSessionRef(s.ptr), cp, cOpts,
		)
		if streamPtr == nil {
			h.finish(errStream("failed to create response stream"))
			return
		}
		defer C.FMRelease(unsafe.Pointer(streamPtr))

		token := registerHandle(h)
		defer releaseHandle(token)

		// Watch for ctx cancellation in parallel.
		if ctx != nil {
			cancelDone := make(chan struct{})
			defer close(cancelDone)
			go func() {
				select {
				case <-ctx.Done():
					// The C stream API doesn't expose a cancel handle; we
					// finish the stream with the ctx error and rely on the
					// iteration goroutine to wind down. The native side will
					// keep producing snapshots until it completes, which is
					// the same behavior as the Python SDK.
					h.finish(ctx.Err())
					// Give the iteration a brief window to notice.
					time.Sleep(50 * time.Millisecond)
				case <-cancelDone:
				}
			}()
		}

			C.fm_stream_iterate(streamPtr, token)
			<-h.done
		}()

	return snapshots, errs
}

type streamErr string

func (e streamErr) Error() string { return string(e) }

func errStream(msg string) error { return streamErr(msg) }
