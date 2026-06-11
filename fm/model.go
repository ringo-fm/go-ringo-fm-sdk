package fm

/*
#include <stdlib.h>
#include "FoundationModels.h"
*/
import "C"

import (
	"runtime"
	"unsafe"
)

// UseCase optimizes the system foundation model for a category of work.
type UseCase int

const (
	UseCaseGeneral        UseCase = 0
	UseCaseContentTagging UseCase = 1
)

// Guardrails sets the safety filtering level on a system foundation model.
type Guardrails int

const (
	GuardrailsDefault                          Guardrails = 0
	GuardrailsPermissiveContentTransformations Guardrails = 1
)

// UnavailableReason explains why the system model can't be used right now.
type UnavailableReason int

const (
	ReasonAppleIntelligenceNotEnabled UnavailableReason = 0
	ReasonDeviceNotEligible           UnavailableReason = 1
	ReasonModelNotReady               UnavailableReason = 2
	ReasonUnknown                     UnavailableReason = 0xFF
)

func (r UnavailableReason) String() string {
	switch r {
	case ReasonAppleIntelligenceNotEnabled:
		return "Apple Intelligence not enabled"
	case ReasonDeviceNotEligible:
		return "device not eligible"
	case ReasonModelNotReady:
		return "model not ready"
	default:
		return "unknown"
	}
}

// SystemLanguageModel is the on-device foundation model that powers Apple
// Intelligence. The zero value is not usable; construct with NewSystemLanguageModel.
type SystemLanguageModel struct {
	ptr unsafe.Pointer // FMSystemLanguageModelRef
}

// ModelOption configures NewSystemLanguageModel.
type ModelOption func(*modelOptions)

type modelOptions struct {
	useCase    UseCase
	guardrails Guardrails
}

// WithUseCase sets the model's use case.
func WithUseCase(uc UseCase) ModelOption {
	return func(o *modelOptions) { o.useCase = uc }
}

// WithGuardrails sets the model's guardrail level.
func WithGuardrails(g Guardrails) ModelOption {
	return func(o *modelOptions) { o.guardrails = g }
}

// NewSystemLanguageModel creates a system language model. Defaults to
// general use case with default guardrails.
func NewSystemLanguageModel(opts ...ModelOption) *SystemLanguageModel {
	o := modelOptions{useCase: UseCaseGeneral, guardrails: GuardrailsDefault}
	for _, opt := range opts {
		opt(&o)
	}
	ptr := C.FMSystemLanguageModelCreate(C.FMSystemLanguageModelUseCase(o.useCase), C.FMSystemLanguageModelGuardrails(o.guardrails))
	m := &SystemLanguageModel{ptr: unsafe.Pointer(ptr)}
	runtime.SetFinalizer(m, (*SystemLanguageModel).release)
	return m
}

// DefaultSystemLanguageModel returns the default system model. Equivalent to
// NewSystemLanguageModel() with no options.
func DefaultSystemLanguageModel() *SystemLanguageModel {
	ptr := C.FMSystemLanguageModelGetDefault()
	m := &SystemLanguageModel{ptr: unsafe.Pointer(ptr)}
	runtime.SetFinalizer(m, (*SystemLanguageModel).release)
	return m
}

// IsAvailable reports whether the model can be used on the current device.
// When the model is unavailable the second return is the reason.
func (m *SystemLanguageModel) IsAvailable() (bool, UnavailableReason) {
	var reason C.FMSystemLanguageModelUnavailableReason
	ok := C.FMSystemLanguageModelIsAvailable(C.FMSystemLanguageModelRef(m.ptr), &reason)
	if bool(ok) {
		return true, ReasonUnknown
	}
	return false, UnavailableReason(reason)
}

// Close releases the underlying C resources. Calling Close more than once is
// safe; the finalizer will not double-release.
func (m *SystemLanguageModel) Close() {
	m.release()
	runtime.SetFinalizer(m, nil)
}

func (m *SystemLanguageModel) release() {
	if m == nil || m.ptr == nil {
		return
	}
	C.FMRelease(m.ptr)
	m.ptr = nil
}
