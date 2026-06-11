package fm

import "testing"

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
