package fm

/*
#include <stdlib.h>
#include "FoundationModels.h"
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"math"
	"unsafe"
)

// FeedbackSentiment describes the user's overall judgment of a response.
type FeedbackSentiment int

const (
	FeedbackSentimentNone FeedbackSentiment = iota
	FeedbackSentimentPositive
	FeedbackSentimentNegative
	FeedbackSentimentNeutral
)

// FeedbackIssueCategory describes the kind of issue being reported.
type FeedbackIssueCategory string

const (
	FeedbackIssueUnhelpful                    FeedbackIssueCategory = "unhelpful"
	FeedbackIssueTooVerbose                   FeedbackIssueCategory = "tooVerbose"
	FeedbackIssueDidNotFollowInstructions     FeedbackIssueCategory = "didNotFollowInstructions"
	FeedbackIssueIncorrect                    FeedbackIssueCategory = "incorrect"
	FeedbackIssueStereotypeOrBias             FeedbackIssueCategory = "stereotypeOrBias"
	FeedbackIssueSuggestiveOrSexual           FeedbackIssueCategory = "suggestiveOrSexual"
	FeedbackIssueVulgarOrOffensive            FeedbackIssueCategory = "vulgarOrOffensive"
	FeedbackIssueTriggeredGuardrailUnexpected FeedbackIssueCategory = "triggeredGuardrailUnexpectedly"
)

// FeedbackIssue provides a categorized reason for feedback.
type FeedbackIssue struct {
	Category    FeedbackIssueCategory `json:"category"`
	Explanation string                `json:"explanation,omitempty"`
}

// FeedbackAttachmentOptions configures a feedback attachment.
type FeedbackAttachmentOptions struct {
	Sentiment           FeedbackSentiment
	Issues              []FeedbackIssue
	DesiredResponseText string
}

// LogFeedbackAttachment returns a FoundationModels feedback attachment payload.
func (s *Session) LogFeedbackAttachment(options FeedbackAttachmentOptions) ([]byte, error) {
	if !options.Sentiment.valid() {
		return nil, fmt.Errorf("feedback attachment: unknown sentiment %d", options.Sentiment)
	}
	for _, issue := range options.Issues {
		if !issue.Category.valid() {
			return nil, fmt.Errorf("feedback attachment: unknown issue category %q", issue.Category)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ptr == nil {
		return nil, fmt.Errorf("session: closed")
	}

	var cIssues *C.char
	if len(options.Issues) > 0 {
		payload, err := json.Marshal(options.Issues)
		if err != nil {
			return nil, err
		}
		cIssues = C.CString(string(payload))
		defer C.free(unsafe.Pointer(cIssues))
	}

	var cDesired *C.char
	if options.DesiredResponseText != "" {
		cDesired = C.CString(options.DesiredResponseText)
		defer C.free(unsafe.Pointer(cDesired))
	}

	var length C.size_t
	var code C.int
	var desc *C.char
	ptr := C.FMLanguageModelSessionLogFeedbackAttachment(
		C.FMLanguageModelSessionRef(s.ptr),
		C.FMFeedbackSentiment(options.Sentiment),
		cIssues,
		cDesired,
		&length,
		&code,
		&desc,
	)
	if ptr == nil {
		return nil, errorFromStatus(GenerationErrorCode(code), goStringAndFree(desc))
	}
	defer C.FMFreeString(ptr)
	if length > C.size_t(math.MaxInt32) {
		return nil, fmt.Errorf("feedback attachment: payload too large")
	}
	return C.GoBytes(unsafe.Pointer(ptr), C.int(length)), nil
}

func (s FeedbackSentiment) valid() bool {
	return s >= FeedbackSentimentNone && s <= FeedbackSentimentNeutral
}

func (c FeedbackIssueCategory) valid() bool {
	switch c {
	case FeedbackIssueUnhelpful,
		FeedbackIssueTooVerbose,
		FeedbackIssueDidNotFollowInstructions,
		FeedbackIssueIncorrect,
		FeedbackIssueStereotypeOrBias,
		FeedbackIssueSuggestiveOrSexual,
		FeedbackIssueVulgarOrOffensive,
		FeedbackIssueTriggeredGuardrailUnexpected:
		return true
	default:
		return false
	}
}
