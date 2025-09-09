# Pinboard CLI Tool

A command-line tool for exercising the Pinboard API client implementation.

## Installation

Build from source:
```sh
make bin/pinboard
# or
go build -o bin/pinboard ./cmd/pinboard/
```

## Configuration

Set up credentials using one of these methods:

### Environment Variables
```sh
export PINBOARD_USERNAME="your_username"
export PINBOARD_TOKEN="your_api_token"
```

### Configuration File
Create `~/.config/hbt/credentials.json`:
```json
{
  "pinboard": {
    "username": "your_username",
    "token": "your_api_token"
  }
}
```

## Usage

### Posts Commands

```sh
# List recent posts
pinboard posts recent --count 10

# List all posts with filtering
pinboard posts list --tags "web,programming" --results 50

# Add a bookmark
pinboard posts add https://example.com "Example Title" --tags "web,demo" --extended "Useful example site"

# Delete a bookmark
pinboard posts delete https://example.com

# Get specific posts
pinboard posts get --url https://example.com
pinboard posts get --tags "web" --date "2024-01-01"

# Get post counts by date
pinboard posts dates --tags "programming"

# Check last update time
pinboard posts update

# Get tag suggestions for a URL
pinboard posts suggest https://github.com/some/repo
```

### Tags Commands

```sh
# List all tags with usage counts
pinboard tags list

# Rename a tag
pinboard tags rename "old-tag" "new-tag"

# Delete a tag
pinboard tags delete "unused-tag"
```

### User Commands

```sh
# Get your API token
pinboard user token

# Get your RSS secret key
pinboard user secret
```

### Notes Commands

```sh
# List all notes
pinboard notes list

# Get a specific note by ID
pinboard notes get "note_id_here"
```

## Output

All data commands output JSON to stdout, making it easy to pipe to tools like `jq`:

```sh
# Pretty-print recent posts
pinboard posts recent | jq .

# Count total bookmarks
pinboard posts list | jq length

# Extract just URLs
pinboard posts recent | jq -r '.[].href'

# Get tags with more than 10 uses
pinboard tags list | jq 'to_entries | map(select(.value > 10))'
```

## Examples

```sh
# Backup all bookmarks to a file
pinboard posts list > my-bookmarks.json

# Add a bookmark with all options
pinboard posts add \
  "https://news.ycombinator.com" \
  "Hacker News" \
  --extended "Tech news and discussion" \
  --tags "news,tech,programming" \
  --private

# Check when you last updated your bookmarks
pinboard posts update

# Find bookmarks from a specific date
pinboard posts get --date "2024-01-15"
```
