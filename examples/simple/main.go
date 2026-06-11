// Simple inference example — Go port of examples/simple_inference.py.
package main

import (
	"context"
	"fmt"
	"log"

	fm "github.com/ringo-fm/go-ringo-fm-sdk/fm"
)

func main() {
	fmt.Println("=== Simple Inference Example ===")
	fmt.Println()

	model := fm.NewSystemLanguageModel()
	defer model.Close()

	ok, reason := model.IsAvailable()
	if !ok {
		fmt.Printf("Model not available: %s\n", reason)
		return
	}

	session, err := fm.NewSession(fm.WithInstructions("You are a helpful assistant that provides concise answers."))
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	ctx := context.Background()

	prompt := "What is the capital of France?"
	fmt.Printf("User: %s\n", prompt)
	resp, err := session.Respond(ctx, fm.TextPrompt(prompt))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Assistant: %s\n\n", resp)

	followUp := "What is its population?"
	fmt.Printf("User: %s\n", followUp)
	resp, err = session.Respond(ctx, fm.TextPrompt(followUp))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Assistant: %s\n\n", resp)
}
