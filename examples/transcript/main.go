// Transcript export/resume example — Go port of examples/transcript_processing.py.
package main

import (
	"context"
	"fmt"
	"log"

	fm "github.com/f4ah6o/go-ringo-fm-sdk/fm"
)

func main() {
	fmt.Println("=== Transcript Example ===")
	fmt.Println()

	model := fm.NewSystemLanguageModel()
	defer model.Close()
	if ok, reason := model.IsAvailable(); !ok {
		fmt.Printf("Model not available: %s\n", reason)
		return
	}

	session, err := fm.NewSession(fm.WithInstructions("Be brief."))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	if _, err := session.Respond(ctx, fm.TextPrompt("Name three rivers in Europe.")); err != nil {
		log.Fatal(err)
	}

	jsonBytes, err := session.Transcript().MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Saved transcript (%d bytes)\n", len(jsonBytes))
	session.Close()

	// Resume from the saved transcript.
	loaded, err := fm.TranscriptFromJSON(jsonBytes)
	if err != nil {
		log.Fatal(err)
	}
	resumed, err := fm.SessionFromTranscript(loaded)
	if err != nil {
		log.Fatal(err)
	}
	defer resumed.Close()

	resp, err := resumed.Respond(ctx, fm.TextPrompt("Now name three more, different ones."))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Resumed response: %s\n", resp)
}
