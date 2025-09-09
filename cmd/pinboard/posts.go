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
	flagTags := fs.String("tags", "", "Filter by comma-separated tags (up to 3)")
	flagStart := fs.Int("start", 0, "Offset for results")
	flagResults := fs.Int("results", 0, "Limit number of results")
	flagFrom := fs.String("from", "", "From date (YYYY-MM-DD)")
	flagTo := fs.String("to", "", "To date (YYYY-MM-DD)")
	flagMeta := fs.Bool("meta", false, "Include metadata")

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
		Meta: *flagMeta,
	}

	if *flagTags != "" {
		opts.Tag = strings.Split(*flagTags, ",")
	}

	if *flagStart > 0 {
		opts.Start = *flagStart
	}

	if *flagResults > 0 {
		opts.Results = *flagResults
	}

	if *flagFrom != "" {
		if fromTime, err := time.Parse("2006-01-02", *flagFrom); err == nil {
			opts.FromDt = fromTime
		} else {
			fmt.Fprintf(os.Stderr, "Error: Invalid from date format. Use YYYY-MM-DD\n")
			os.Exit(1)
		}
	}

	if *flagTo != "" {
		if toTime, err := time.Parse("2006-01-02", *flagTo); err == nil {
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
	flagCount := fs.Int("count", 15, "Number of results (max 100)")
	flagTags := fs.String("tags", "", "Filter by comma-separated tags (up to 3)")
	flagMeta := fs.Bool("meta", false, "Include metadata")

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
	if *flagTags != "" {
		tags = strings.Split(*flagTags, ",")
	}

	posts, err := client.GetRecentPosts(context.Background(), *flagCount, tags, *flagMeta)
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
	flagExtended := fs.String("extended", "", "Extended description")
	flagTags := fs.String("tags", "", "Comma-separated tags")
	flagDate := fs.String("date", "", "Date (YYYY-MM-DD)")
	flagReplace := fs.Bool("replace", true, "Replace existing bookmark")
	flagPrivate := fs.Bool("private", false, "Make bookmark private")
	flagToread := fs.Bool("toread", false, "Mark as 'to read'")

	fs.Parse(args[2:])

	client, err := createClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	opts := &pinboard.AddPostOptions{
		Extended: *flagExtended,
		Tags:     *flagTags,
		Replace:  flagReplace,
		ToRead:   flagToread,
	}

	// Shared defaults to true unless explicitly set to private
	shared := !*flagPrivate
	opts.Shared = &shared

	if *flagDate != "" {
		if dt, err := time.Parse("2006-01-02", *flagDate); err == nil {
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
	flagURL := fs.String("url", "", "URL to get")
	flagTags := fs.String("tags", "", "Filter by comma-separated tags (up to 3)")
	flagDate := fs.String("date", "", "Date (YYYY-MM-DD)")
	flagMeta := fs.Bool("meta", false, "Include metadata")

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
	if *flagTags != "" {
		tags = strings.Split(*flagTags, ",")
	}

	posts, err := client.GetPosts(context.Background(), tags, *flagDate, *flagURL, *flagMeta)
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
	flagTags := fs.String("tags", "", "Filter by comma-separated tags")

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
	if *flagTags != "" {
		tags = strings.Split(*flagTags, ",")
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
