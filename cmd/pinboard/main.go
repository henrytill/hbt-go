package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/henrytill/hbt-go/internal/client/pinboard"
)

var (
	Version    = "0.1.0-dev"
	Commit     = "unknown"
	CommitDate = "unknown"
	TreeState  = "unknown"
)

func showVersion() {
	fmt.Printf("pinboard %s\n", Version)
}

func showUsage() {
	fmt.Printf("Usage: %s <subcommand> [options]\n\n", os.Args[0])
	fmt.Println("Pinboard API client for testing and exercising the API")
	fmt.Println("\nSubcommands:")
	fmt.Println("  posts    - Posts operations (list, add, delete, recent, etc.)")
	fmt.Println("  tags     - Tags operations (list, rename, delete)")
	fmt.Println("  user     - User operations (get token, secret)")
	fmt.Println("  notes    - Notes operations (list, get)")
	fmt.Println("  version  - Show version")
	fmt.Println("  help     - Show this help")
	fmt.Println("\nCredentials:")
	fmt.Println("  Set PINBOARD_USERNAME and PINBOARD_TOKEN environment variables")
	fmt.Println("  Or create ~/.config/hbt/credentials.json with:")
	fmt.Println(`  {"pinboard": {"username": "your_username", "token": "your_token"}}`)
	fmt.Println("\nExamples:")
	fmt.Println("  pinboard posts recent --count 5")
	fmt.Println("  pinboard posts add https://example.com \"Example Title\" --tags \"web,demo\"")
	fmt.Println("  pinboard tags list")
	fmt.Println("  pinboard user token")
}

func createClient() (*pinboard.Client, error) {
	client, err := pinboard.NewClientFromCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	return client, nil
}

func outputJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func main() {
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "version", "--version", "-V":
		showVersion()
		return
	case "help", "--help", "-h":
		showUsage()
		return
	case "posts":
		handlePosts(os.Args[2:])
	case "tags":
		handleTags(os.Args[2:])
	case "user":
		handleUser(os.Args[2:])
	case "notes":
		handleNotes(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", subcommand)
		showUsage()
		os.Exit(1)
	}
}

func handlePosts(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Posts subcommand requires an operation\n")
		fmt.Fprintf(os.Stderr, "Available operations: list, recent, add, delete, get, dates, update, suggest\n")
		os.Exit(1)
	}

	operation := args[0]

	switch operation {
	case "list":
		handlePostsList(args[1:])
	case "recent":
		handlePostsRecent(args[1:])
	case "add":
		handlePostsAdd(args[1:])
	case "delete":
		handlePostsDelete(args[1:])
	case "get":
		handlePostsGet(args[1:])
	case "dates":
		handlePostsDates(args[1:])
	case "update":
		handlePostsUpdate(args[1:])
	case "suggest":
		handlePostsSuggest(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown posts operation: %s\n", operation)
		os.Exit(1)
	}
}

func handleTags(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Tags subcommand requires an operation\n")
		fmt.Fprintf(os.Stderr, "Available operations: list, rename, delete\n")
		os.Exit(1)
	}

	operation := args[0]

	switch operation {
	case "list":
		handleTagsList(args[1:])
	case "rename":
		handleTagsRename(args[1:])
	case "delete":
		handleTagsDelete(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown tags operation: %s\n", operation)
		os.Exit(1)
	}
}

func handleUser(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "User subcommand requires an operation\n")
		fmt.Fprintf(os.Stderr, "Available operations: token, secret\n")
		os.Exit(1)
	}

	operation := args[0]

	switch operation {
	case "token":
		handleUserToken(args[1:])
	case "secret":
		handleUserSecret(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown user operation: %s\n", operation)
		os.Exit(1)
	}
}

func handleNotes(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Notes subcommand requires an operation\n")
		fmt.Fprintf(os.Stderr, "Available operations: list, get\n")
		os.Exit(1)
	}

	operation := args[0]

	switch operation {
	case "list":
		handleNotesList(args[1:])
	case "get":
		handleNotesGet(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown notes operation: %s\n", operation)
		os.Exit(1)
	}
}
