package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/henrytill/hbt-go/internal/client/pinboard"
)

func handlePostsList(args []string) {
	fs := flag.NewFlagSet("posts list", flag.ExitOnError)
	tagsFlag := fs.String("tags", "", "Filter by comma-separated tags (up to 3)")
	startFlag := fs.Int("start", 0, "Offset for results")
	resultsFlag := fs.Int("results", 0, "Limit number of results")
	fromFlag := fs.String("from", "", "From date (YYYY-MM-DD)")
	toFlag := fs.String("to", "", "To date (YYYY-MM-DD)")
	metaFlag := fs.Bool("meta", false, "Include metadata")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pinboard posts list [options]\n")
		fmt.Fprintf(os.Stderr, "Get all posts with optional filtering\n\n")
		fs.PrintDefaults()
	}

	fs.Parse(args)

	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	opts := &pinboard.GetAllPostsOptions{
		Meta: *metaFlag,
	}

	if *tagsFlag != "" {
		opts.Tag = strings.Split(*tagsFlag, ",")
	}

	if *startFlag > 0 {
		opts.Start = *startFlag
	}

	if *resultsFlag > 0 {
		opts.Results = *resultsFlag
	}

	if *fromFlag != "" {
		if fromTime, err := time.Parse("2006-01-02", *fromFlag); err == nil {
			opts.FromDt = fromTime
		} else {
			fmt.Fprintf(os.Stderr, "Error: Invalid from date format. Use YYYY-MM-DD\n")
			os.Exit(1)
		}
	}

	if *toFlag != "" {
		if toTime, err := time.Parse("2006-01-02", *toFlag); err == nil {
			opts.ToDt = toTime
		} else {
			fmt.Fprintf(os.Stderr, "Error: Invalid to date format. Use YYYY-MM-DD\n")
			os.Exit(1)
		}
	}

	posts, err := client.GetAllPosts(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := outputJSON(posts); err != nil {
		fmt.Fprintf(os.Stderr, "Error outputting JSON: %v\n", err)
		os.Exit(1)
	}
}

func handlePostsRecent(args []string) {
	fs := flag.NewFlagSet("posts recent", flag.ExitOnError)
	countFlag := fs.Int("count", 15, "Number of results (max 100)")
	tagsFlag := fs.String("tags", "", "Filter by comma-separated tags (up to 3)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pinboard posts recent [options]\n")
		fmt.Fprintf(os.Stderr, "Get recent posts\n\n")
		fs.PrintDefaults()
	}

	fs.Parse(args)

	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var tags []string
	if *tagsFlag != "" {
		tags = strings.Split(*tagsFlag, ",")
	}

	posts, err := client.GetRecentPosts(context.Background(), *countFlag, tags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := outputJSON(posts); err != nil {
		fmt.Fprintf(os.Stderr, "Error outputting JSON: %v\n", err)
		os.Exit(1)
	}
}

func handlePostsAdd(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: pinboard posts add <url> <title> [options]\n")
		fmt.Fprintf(os.Stderr, "Add a bookmark\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		fmt.Fprintf(os.Stderr, "  --extended <text>    Extended description\n")
		fmt.Fprintf(os.Stderr, "  --tags <tags>        Comma-separated tags\n")
		fmt.Fprintf(os.Stderr, "  --date <date>        Date (YYYY-MM-DD)\n")
		fmt.Fprintf(os.Stderr, "  --replace=false      Don't replace existing bookmark\n")
		fmt.Fprintf(os.Stderr, "  --private            Make bookmark private\n")
		fmt.Fprintf(os.Stderr, "  --toread             Mark as 'to read'\n")
		os.Exit(1)
	}

	url := args[0]
	title := args[1]

	fs := flag.NewFlagSet("posts add", flag.ExitOnError)
	extendedFlag := fs.String("extended", "", "Extended description")
	tagsFlag := fs.String("tags", "", "Comma-separated tags")
	dateFlag := fs.String("date", "", "Date (YYYY-MM-DD)")
	replaceFlag := fs.Bool("replace", true, "Replace existing bookmark")
	privateFlag := fs.Bool("private", false, "Make bookmark private")
	toreadFlag := fs.Bool("toread", false, "Mark as 'to read'")

	fs.Parse(args[2:])

	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	opts := &pinboard.AddPostOptions{
		Extended: *extendedFlag,
		Tags:     *tagsFlag,
		Replace:  replaceFlag,
		ToRead:   toreadFlag,
	}

	// Shared defaults to true unless explicitly set to private
	shared := !*privateFlag
	opts.Shared = &shared

	if *dateFlag != "" {
		if dt, err := time.Parse("2006-01-02", *dateFlag); err == nil {
			opts.Dt = dt
		} else {
			fmt.Fprintf(os.Stderr, "Error: Invalid date format. Use YYYY-MM-DD\n")
			os.Exit(1)
		}
	}

	err = client.AddPost(context.Background(), url, title, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Bookmark added successfully")
}

func handlePostsDelete(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: pinboard posts delete <url>\n")
		os.Exit(1)
	}

	url := args[0]

	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = client.DeletePost(context.Background(), url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Bookmark deleted successfully")
}

func handlePostsGet(args []string) {
	fs := flag.NewFlagSet("posts get", flag.ExitOnError)
	urlFlag := fs.String("url", "", "URL to get")
	tagsFlag := fs.String("tags", "", "Filter by comma-separated tags (up to 3)")
	dateFlag := fs.String("date", "", "Date (YYYY-MM-DD)")
	metaFlag := fs.Bool("meta", false, "Include metadata")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pinboard posts get [options]\n")
		fmt.Fprintf(os.Stderr, "Get specific posts\n\n")
		fs.PrintDefaults()
	}

	fs.Parse(args)

	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var tags []string
	if *tagsFlag != "" {
		tags = strings.Split(*tagsFlag, ",")
	}

	posts, err := client.GetPosts(context.Background(), tags, *dateFlag, *urlFlag, *metaFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := outputJSON(posts); err != nil {
		fmt.Fprintf(os.Stderr, "Error outputting JSON: %v\n", err)
		os.Exit(1)
	}
}

func handlePostsDates(args []string) {
	fs := flag.NewFlagSet("posts dates", flag.ExitOnError)
	tagsFlag := fs.String("tags", "", "Filter by comma-separated tags")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pinboard posts dates [options]\n")
		fmt.Fprintf(os.Stderr, "Get post counts by date\n\n")
		fs.PrintDefaults()
	}

	fs.Parse(args)

	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var tags []string
	if *tagsFlag != "" {
		tags = strings.Split(*tagsFlag, ",")
	}

	dates, err := client.GetPostsDates(context.Background(), tags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := outputJSON(dates); err != nil {
		fmt.Fprintf(os.Stderr, "Error outputting JSON: %v\n", err)
		os.Exit(1)
	}
}

func handlePostsUpdate(_ []string) {
	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	updateTime, err := client.GetUpdate(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result := map[string]string{
		"update_time": updateTime.Format(time.RFC3339),
	}

	if err := outputJSON(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error outputting JSON: %v\n", err)
		os.Exit(1)
	}
}

func handlePostsSuggest(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: pinboard posts suggest <url>\n")
		os.Exit(1)
	}

	url := args[0]

	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	popular, recommended, err := client.SuggestTags(context.Background(), url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result := map[string][]string{
		"popular":     popular,
		"recommended": recommended,
	}

	if err := outputJSON(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error outputting JSON: %v\n", err)
		os.Exit(1)
	}
}
