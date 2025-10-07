package main

import (
	"context"
	"fmt"
	"os"
)

func handleTagsList(_ []string) {
	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	tags, err := client.GetTags(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := outputJSON(tags); err != nil {
		fmt.Fprintf(os.Stderr, "Error outputting JSON: %v\n", err)
		os.Exit(1)
	}
}

func handleTagsRename(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: pinboard tags rename <old_tag> <new_tag>\n")
		os.Exit(1)
	}

	oldTag := args[0]
	newTag := args[1]

	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = client.RenameTag(context.Background(), oldTag, newTag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Tag renamed from '%s' to '%s' successfully\n", oldTag, newTag)
}

func handleTagsDelete(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: pinboard tags delete <tag>\n")
		os.Exit(1)
	}

	tag := args[0]

	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = client.DeleteTag(context.Background(), tag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Tag '%s' deleted successfully\n", tag)
}
