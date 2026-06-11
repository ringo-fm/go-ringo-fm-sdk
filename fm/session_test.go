package fm

import "testing"

func TestSessionLifecycle(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if s.IsResponding() {
		t.Fatal("new session should not be responding")
	}
	if s.Transcript() == nil {
		t.Fatal("transcript should not be nil")
	}
}

func TestSessionPrewarm(t *testing.T) {
	s, err := NewSession(WithInstructions("You are a helpful assistant."))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Prewarm is a fire-and-forget hint; both forms must be safe and must not
	// flip the session into a responding state.
	s.Prewarm("")
	s.Prewarm("Summarize the following text:")
	if s.IsResponding() {
		t.Fatal("prewarm should not mark the session as responding")
	}
}

func TestSessionPrewarmAfterClose(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatal(err)
	}
	s.Close()

	// Prewarm on a closed session must be a no-op, not a crash.
	s.Prewarm("ignored")
}
