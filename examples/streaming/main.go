// Streaming example — Go port of examples/streaming_example.py.
package main

import (
	"context"
	"fmt"
	"log"

	fm "github.com/f4ah6o/go-ringo-fm-sdk/fm"
)

func main() {
	fmt.Println("=== Streaming Example ===")
	fmt.Println()

	model := fm.NewSystemLanguageModel()
	defer model.Close()
	if ok, reason := model.IsAvailable(); !ok {
		fmt.Printf("Model not available: %s\n", reason)
		return
	}

	session, err := fm.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	prompt := fm.TextPrompt("Tell me a one-paragraph story about a curious cat.")
	snapshots, errs := session.StreamResponse(context.Background(), prompt)

	prev := ""
	for snap := range snapshots {
		// snapshots are cumulative; print only the new tail.
		if len(snap) > len(prev) {
			fmt.Print(snap[len(prev):])
			prev = snap
		}
	}
	if err := <-errs; err != nil {
		log.Fatalf("\nstream error: %v", err)
	}
	fmt.Println()
}
