package fm

import (
	"errors"
	"fmt"
)

// GenerationErrorCode mirrors the C-side status codes returned by the
// FoundationModels callbacks. Code values must stay in sync with
// FoundationModels.h's status enums.
type GenerationErrorCode int32

const (
	codeSuccess                     GenerationErrorCode = 0
	codeExceededContextWindowSize   GenerationErrorCode = 1
	codeAssetsUnavailable           GenerationErrorCode = 2
	codeGuardrailViolation          GenerationErrorCode = 3
	codeUnsupportedGuide            GenerationErrorCode = 4
	codeUnsupportedLanguageOrLocale GenerationErrorCode = 5
	codeDecodingFailure             GenerationErrorCode = 6
	codeRateLimited                 GenerationErrorCode = 7
	codeConcurrentRequests          GenerationErrorCode = 8
	codeRefusal                     GenerationErrorCode = 9
	codeInvalidSchema               GenerationErrorCode = 10
	codeUnknownError                GenerationErrorCode = 0xFF
)

// Sentinel errors. Use errors.Is to check.
var (
	ErrFoundationModels           = errors.New("foundation models error")
	ErrGeneration                 = fmt.Errorf("%w: generation", ErrFoundationModels)
	ErrExceededContextWindowSize  = fmt.Errorf("%w: context window size exceeded", ErrGeneration)
	ErrAssetsUnavailable          = fmt.Errorf("%w: required assets are unavailable", ErrGeneration)
	ErrGuardrailViolation         = fmt.Errorf("%w: guardrail violation", ErrGeneration)
	ErrUnsupportedGuide           = fmt.Errorf("%w: unsupported guide", ErrGeneration)
	ErrUnsupportedLanguageOrLocale = fmt.Errorf("%w: unsupported language or locale", ErrGeneration)
	ErrDecodingFailure            = fmt.Errorf("%w: decoding failure", ErrGeneration)
	ErrRateLimited                = fmt.Errorf("%w: rate limited", ErrGeneration)
	ErrConcurrentRequests         = fmt.Errorf("%w: too many concurrent requests", ErrGeneration)
	ErrInvalidSchema              = fmt.Errorf("%w: invalid generation schema", ErrFoundationModels)
)

// RefusalError is returned when the model refuses to produce content. It
// wraps ErrGeneration so errors.Is(err, ErrGeneration) returns true.
type RefusalError struct {
	Message          string
	DebugDescription string
}

func (e *RefusalError) Error() string {
	if e.DebugDescription != "" {
		return fmt.Sprintf("model refused to generate content: %s (%s)", e.Message, e.DebugDescription)
	}
	return fmt.Sprintf("model refused to generate content: %s", e.Message)
}

func (e *RefusalError) Unwrap() error { return ErrGeneration }

// ToolCallError is returned when a tool's Call method fails.
type ToolCallError struct {
	ToolName string
	Err      error
}

func (e *ToolCallError) Error() string {
	return fmt.Sprintf("tool %q failed: %v", e.ToolName, e.Err)
}

func (e *ToolCallError) Unwrap() error { return e.Err }

// errorFromStatus converts a C status code (plus optional debug description)
// into the appropriate Go error. Returns nil for codeSuccess.
func errorFromStatus(status GenerationErrorCode, debug string) error {
	switch status {
	case codeSuccess:
		return nil
	case codeExceededContextWindowSize:
		return wrapStatus(ErrExceededContextWindowSize, debug)
	case codeAssetsUnavailable:
		return wrapStatus(ErrAssetsUnavailable, debug)
	case codeGuardrailViolation:
		return wrapStatus(ErrGuardrailViolation, debug)
	case codeUnsupportedGuide:
		return wrapStatus(ErrUnsupportedGuide, debug)
	case codeUnsupportedLanguageOrLocale:
		return wrapStatus(ErrUnsupportedLanguageOrLocale, debug)
	case codeDecodingFailure:
		return wrapStatus(ErrDecodingFailure, debug)
	case codeRateLimited:
		return wrapStatus(ErrRateLimited, debug)
	case codeConcurrentRequests:
		return wrapStatus(ErrConcurrentRequests, debug)
	case codeRefusal:
		return &RefusalError{Message: "model refused to generate content", DebugDescription: debug}
	case codeInvalidSchema:
		return wrapStatus(ErrInvalidSchema, debug)
	default:
		if debug != "" {
			return fmt.Errorf("%w: unknown status %d: %s", ErrGeneration, status, debug)
		}
		return fmt.Errorf("%w: unknown status %d", ErrGeneration, status)
	}
}

func wrapStatus(base error, debug string) error {
	if debug == "" {
		return base
	}
	return fmt.Errorf("%w: %s", base, debug)
}
