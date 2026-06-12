package fm

import "testing"

func TestLogFeedbackAttachment(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	payload, err := s.LogFeedbackAttachment(FeedbackAttachmentOptions{
		Sentiment: FeedbackSentimentNegative,
		Issues: []FeedbackIssue{{
			Category:    FeedbackIssueIncorrect,
			Explanation: "Expected a shorter response.",
		}},
		DesiredResponseText: "A shorter desired response.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) == 0 {
		t.Fatal("feedback attachment should not be empty")
	}
}

func TestLogFeedbackAttachmentWithDesiredResponseContent(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	content, err := GeneratedContentFromJSON(`{"answer":"A concise desired response."}`)
	if err != nil {
		t.Fatal(err)
	}
	defer content.Close()

	payload, err := s.LogFeedbackAttachment(FeedbackAttachmentOptions{
		Sentiment:              FeedbackSentimentPositive,
		DesiredResponseContent: content,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) == 0 {
		t.Fatal("feedback attachment should not be empty")
	}
}

func TestLogFeedbackAttachmentRejectsTextAndContent(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	content, err := GeneratedContentFromJSON(`{"answer":"A concise desired response."}`)
	if err != nil {
		t.Fatal(err)
	}
	defer content.Close()

	_, err = s.LogFeedbackAttachment(FeedbackAttachmentOptions{
		Sentiment:              FeedbackSentimentPositive,
		DesiredResponseText:    "A text response.",
		DesiredResponseContent: content,
	})
	if err == nil {
		t.Fatal("expected text/content conflict error")
	}
}

func TestLogFeedbackAttachmentRejectsUnknownIssueCategory(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	_, err = s.LogFeedbackAttachment(FeedbackAttachmentOptions{
		Sentiment: FeedbackSentimentNeutral,
		Issues: []FeedbackIssue{{
			Category: FeedbackIssueCategory("notARealCategory"),
		}},
	})
	if err == nil {
		t.Fatal("expected unknown category error")
	}
}

func TestLogFeedbackAttachmentRejectsUnknownSentiment(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	_, err = s.LogFeedbackAttachment(FeedbackAttachmentOptions{
		Sentiment: FeedbackSentiment(99),
	})
	if err == nil {
		t.Fatal("expected unknown sentiment error")
	}
}
