package fm

import (
	"encoding/json"
	"testing"
)

func TestTranscriptEntryCountNewSession(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// A brand-new session transcript has zero entries.
	if got := s.Transcript().EntryCount(); got != 0 {
		t.Fatalf("EntryCount() = %d, want 0 for a fresh session", got)
	}
}

func TestTranscriptEntryCountNilPtr(t *testing.T) {
	// A zero-value Transcript (nil sessionPtr) must return 0, not crash.
	tr := &Transcript{}
	if got := tr.EntryCount(); got != 0 {
		t.Fatalf("EntryCount() on zero Transcript = %d, want 0", got)
	}
}

func TestSessionFromTranscriptRoundTrip(t *testing.T) {
	orig, err := NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer orig.Close()

	// Serialise the empty-session transcript.
	data, err := json.Marshal(orig.Transcript())
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("transcript JSON is empty")
	}

	// Load it back as a Transcript.
	tr, err := TranscriptFromJSON(data)
	if err != nil {
		t.Fatalf("TranscriptFromJSON: %v", err)
	}

	// Restore a session from the transcript.
	restored, err := SessionFromTranscript(tr)
	if err != nil {
		t.Fatalf("SessionFromTranscript: %v", err)
	}
	defer restored.Close()

	if restored.IsResponding() {
		t.Fatal("restored session should not be responding")
	}
}
