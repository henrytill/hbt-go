package main

import (
	"context"
	"fmt"
	"os"
)

func handleUserToken(_ []string) {
	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	token, err := client.GetAPIToken(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(token)
}

func handleUserSecret(_ []string) {
	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	secret, err := client.GetSecret(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(secret)
}

func handleNotesList(_ []string) {
	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	notes, err := client.ListNotes(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := outputJSON(notes); err != nil {
		fmt.Fprintf(os.Stderr, "Error outputting JSON: %v\n", err)
		os.Exit(1)
	}
}

func handleNotesGet(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: pinboard notes get <note_id>\n")
		os.Exit(1)
	}

	noteID := args[0]

	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	note, err := client.GetNote(context.Background(), noteID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := outputJSON(note); err != nil {
		fmt.Fprintf(os.Stderr, "Error outputting JSON: %v\n", err)
		os.Exit(1)
	}
}
