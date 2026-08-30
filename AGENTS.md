# hbt-go

## REMEMBER

**Use GitHub MCP for all GitHub queries** (instead of fetching webpages)

**Use `go doc` for viewing documentation at the CLI**

**Never work directly on `master`** - branch first, land via PR (see [Git Workflow](#git-workflow))

**Add new non-test `internal/**` files to `SOURCES` in the GNUmakefile** - the build will not see them otherwise (`cmd/` files are picked up by the pattern rule)

## Overview

A Go implementation of the hbt (hierarchical bookmark tool) as part of a differential implementation strategy alongside:
- [hbt-rs](https://github.com/henrytill/hbt-rs) (Rust)
- [hbt-ocaml](https://github.com/henrytill/hbt-ocaml) (OCaml)
- [hbt-hs](https://github.com/henrytill/hbt-hs) (Haskell)

The tool processes bookmark files from various sources (Pinboard exports, browser bookmarks, Markdown files) and converts them to different output formats while maintaining hierarchical relationships and metadata.

Two binaries are built into `bin/`:
- `hbt` - the format conversion tool
- `pinboard` - a Pinboard API client for testing and exercising the live API

Language-agnostic test data for conformance testing:
- [hbt-data](https://github.com/henrytill/hbt-data) - imported as git submodule in `test/testdata/`

## Architecture

### Core Components

**Data Model**: A `Collection` is a graph of bookmark entities held as parallel slices rather than linked nodes.

- `Collection` - `entities []Entity`, `edges [][]uint`, and a `urls map[string]uint` index for deduplication
- `Id` - an opaque handle pairing an owning `*Collection` with an index; `checkId` rejects ids from a different collection
- `Entity` - a bookmark: `URI`, `CreatedAt`, `UpdatedAt Set[UpdatedAt]`, `Names`, `Labels`, `Shared`, `ToRead`, `IsFeed`, `Extended`, `LastVisitedAt`

**Tri-state fields**: `Shared`, `ToRead`, and `IsFeed` wrap an unexported `optBool` (set/unset plus value) so that "absent" is distinct from "false". Each exposes `Get() (bool, bool)` and a `Merge` that combines two values. `LastVisitedAt` is optional too, but as a timestamp plus a `Valid` flag rather than an `optBool` wrapper.

**Timestamps**: `CreatedAt`, `UpdatedAt`, and `LastVisitedAt` are distinct types over an unexported `timestamp`, a Unix second count -- the resolution the wire format carries, and what they are constructed from. This mirrors hbt-rs, where all three are newtypes over one `Time`. The doc comment on `timestamp` says why seconds rather than `time.Time`.

**Set idiom**: `Set[T comparable] map[T]struct{}` in `internal/types/set.go`, with `NewSet`, `Add`, `Merge`, and `SortedSlice` for deterministic output. `Add` and `Merge` return the set, since a nil map cannot be assigned into.

**Interface-Based Design**: Clean separation between parsing and formatting, both defined in `internal/types/intf.go`:
- `Parser` - `Parse(r io.Reader) (Collection, error)`
- `Formatter` - `Format(w io.Writer, coll *Collection) error`

**Format registry** (`internal/formats.go`): each `Format` carries a name and a `FormatCapability` bitmask (`CapInput`, `CapOutput`, `CapBoth`). Parsers and formatters are looked up in `map[Format]types.Parser` / `map[Format]types.Formatter`, so adding a format means adding one `Format` value plus its registry entry.

| Format | Input | Output |
| --- | --- | --- |
| `json` (Pinboard) | yes | no |
| `xml` (Pinboard) | yes | no |
| `markdown` | yes | no |
| `html` (Netscape / browser export) | yes | yes |
| `yaml` | no | yes |

### Package Structure

```
cmd/
├── hbt/             # Format conversion CLI
└── pinboard/        # Pinboard API client CLI (posts, tags, user, notes)
internal/
├── types/           # Core types, interfaces, and Collection logic
├── pinboard/        # Shared Pinboard wire types (Post, Posts, Note)
├── parser/          # Input format parsers
│   └── pinboard/    # Pinboard JSON and XML parsers
├── formatter/       # Output format formatters (html, yaml)
├── client/
│   └── pinboard/    # Pinboard HTTP API client
├── belnap/          # Four-valued logic values and packed bitvectors
├── formats.go       # Format registry and dispatch logic
└── mappings.go      # Label transformation system
test/
├── testgen.go       # go:generate source (//go:build ignore); writes cli_test.go
├── testgen_deps.go  # Blank imports keeping testgen.go's deps in the module graph
├── cli_test.go      # Generated conformance tests (committed)
├── cli_flags_test.go
├── testutil.go      # Shared test helpers
└── testdata/        # hbt-data submodule
```

**Dependency Flow**: Clean acyclic tree with `internal/types` as root, ensuring no circular dependencies. `internal/pinboard` holds the wire types so that both `internal/types` and the parsers can use them without a cycle.

**`internal/belnap` is standalone** - nothing else in the tree imports it. It is exploratory work toward representing contradictory metadata across sources, not part of the conversion pipeline.

## Design Patterns

### Semantic Versioning
- `Version` type wrapping `golang.org/x/mod/semver` for validation and comparison
- Compatibility checking during deserialization with meaningful error messages
- User-friendly serialization format (`0.1.0`) while using proper semver internally (`v0.1.0`)
- 0.x.x development rules: exact major.minor compatibility required (breaking changes allowed in minor versions)

### Graph Operations
- `AddEdges()` creates edges in both directions, following the Haskell implementation pattern, and deduplicates
- No unnecessary sorting of edge lists (node IDs don't require ordering for functionality)

### Parser Architecture
- Stateless parser structs with local state for each parsing operation
- Direct domain type creation (Entity, Collection) rather than intermediate representations
- URL deduplication via `Collection.Upsert()`, which merges into an existing entity through the unexported `absorb()` when the URI is already present

### Serialization
- `Collection` and `Entity` convert through unexported `collectionRepr` / `entityRepr` shapes rather than tagging the domain types
- `MarshalYAML`/`UnmarshalYAML` and `MarshalJSON`/`UnmarshalJSON` are thin wrappers over `toRepr`/`fromRepr`

## Implementation Notes

### Memory Efficiency
- `*url.URL` pointers for larger structs to avoid copying

### Error Handling
- Validation happens immediately at parsing boundaries rather than being deferred
- Graceful handling of malformed input (empty files, invalid URLs, etc.)

### Pinboard API Client
- Auth via `AuthMethod` interface: `BasicAuth` or `TokenAuth` (`auth_token=user:TOKEN` query parameter)
- Credentials come from `PINBOARD_USERNAME`/`PINBOARD_TOKEN`, else `hbt/credentials.json` under `os.UserConfigDir()` (`$XDG_CONFIG_HOME` or `~/.config` on Linux, `~/Library/Application Support` on macOS). The file needs a top-level `pinboard` object: `{"pinboard": {"username": ..., "token": ...}}`
- Rate limits are enforced client-side: 3s between requests, 5m for `posts/all`, 1m for `posts/recent`
- 429 responses are retried up to 3 times with an exponential backoff that honors `Retry-After`

### Testing Strategy
- `test/cli_test.go` is **generated** by `go generate ./test` from the hbt-data submodule and committed; edit `test/testgen.go`, never the generated file
- Golden file testing for output format validation, covering all format combinations
- All parsers and formatters exercised through CLI integration, plus unit tests alongside each package
- The API client is tested against an `httptest` server; no tests hit the live Pinboard API

## Git Workflow

**Never commit to `master`.** All work happens on a feature branch, which lands via pull request.

```sh
git checkout -b <topic>   # branch first, before making any changes
# ... work, commit ...
git push -u origin <topic>
gh pr create
gh pr merge --rebase      # after CI is green
```

If changes have already been made on `master` by mistake, move them to a branch (`git checkout -b <topic>`) before committing rather than pushing.

### Branch protection on `origin/master`

The GitHub remote enforces this; direct pushes to `master` will be rejected.

| Rule | Setting |
| --- | --- |
| Pull request required | yes, 0 approvals (stale reviews dismissed) |
| Required status checks | `Linux (go)`, `Linux (Nix flake)`, strict (branch must be up to date) |
| Linear history | required |
| Conversation resolution | required |
| Force pushes / deletions | blocked |
| Applies to administrators | yes (no bypass) |

### Merge settings

Rebase is the only merge method enabled; squash merges and merge commits are turned off. Merged branches are deleted automatically on GitHub, so prune locally afterwards:

```sh
git fetch --prune
git branch -D <topic>   # -d may refuse: rebase merges rewrite SHAs
```

Rebase merges always create new commit SHAs, so a local branch kept after merging will look diverged from `master`. Delete it rather than reusing it.

### The `ivan` remote

`ivan:/srv/git/hbt-go.git` is a plain bare repo with no protection. Pushing `master` there directly is fine and unaffected by the above.

## Development Guidelines

### Go Style Preferences
Following idiomatic Go practices with influences from:
- [Google Go Style Guide](https://google.github.io/styleguide/go/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- High-quality Go projects (Tailscale, CUE, Starlark-Go)

**When conflicts arise, prefer go.dev documentation over other sources.**

### Code Conventions
- **No comments unless requested** - self-documenting code preferred
- **Early return patterns** for guard clauses and error handling
- **Interface-based design** for extensibility
- **Minimal dependencies** - prefer stdlib and golang.org packages over third-party

### Build System

`GNUmakefile`, driving `go build` per binary:

| Target | Effect |
| --- | --- |
| `all` | Build `bin/hbt` and `bin/pinboard` (`CGO_ENABLED=0`) |
| `test` | `go generate ./test` then `go test -v ./...` (builds the binaries first) |
| `lint` | `go vet`, `staticcheck`, `deadcode -test` |
| `fmt` / `fix` | `go fmt ./...` / `go fix ./...` |
| `tags` / `TAGS` | ctags-universal indices over `SOURCES` |
| `clean` / `distclean` | Remove binaries / also remove `bin/` |

`SOURCES` lists every non-test file under `internal/` and doubles as a prerequisite list and the ctags input. **A new `internal/` file must be added there** or it will neither trigger a rebuild nor appear in the tag indices. Per-binary sources come from the pattern rule's `cmd/%/*.go` prerequisite instead, so files under `cmd/` must *not* be added to `SOURCES` - doing so would make each binary depend on the other's sources.

Nix flake (`flake.nix`) provides the second CI job; `make test` and the flake build must both pass before a PR can merge.

## Dependencies

### Core Dependencies
- `golang.org/x/mod/semver` - Official Go semantic versioning (preferred over third-party)
- `golang.org/x/net/html` - HTML parsing (official extended library)
- `golang.org/x/text` - Unicode-aware title casing, used only by `test/testgen.go`. Because that file is `//go:build ignore`, the module graph cannot see the import; `test/testgen_deps.go` blank-imports `x/text/cases` and `x/text/language` to hold it in `go.mod`. Deleting it as dead code drops the dependency and breaks `go generate ./test`.
- `github.com/goccy/go-yaml` - YAML processing (high-performance, well-maintained)
- `github.com/yuin/goldmark` - CommonMark/Markdown parsing (standard library quality)

Go version is pinned by `go.mod` (`go 1.25.0`); CI reads it via `go-version-file`.

### Dependency Philosophy
1. **Standard library first** - Use built-in Go packages when available
2. **golang.org packages second** - Official extended libraries preferred
3. **Third-party last** - Only when necessary, choose well-maintained options with good APIs

## CLI Interface

### `hbt`

`hbt [OPTIONS] FILE` - matching the Rust/OCaml versions:
- `-f`/`--from` - Input format (auto-detected from file extension)
- `-t`/`--to` - Output format
- `-o` - Output file (defaults to stdout)
- `--info` - Show collection info (entity count)
- `--list-tags` - List all labels/tags
- `--mappings FILE` - Apply label transformations from JSON/YAML file
- `-V`/`--version` - Show version

Format detection runs on both ends when the corresponding flag is absent: input from the argument's extension (`.html`, `.json`, `.xml`, `.md`), output from the `-o` filename (`.html`, `.yaml`). `.yml` is not recognized and errors.

Both binaries carry `Version`, `Commit`, `CommitDate`, and `TreeState` vars, but nothing populates all four outside a release. `.slsa-goreleaser/*.yml` sets all four via `-X main.*`, and builds `./cmd/hbt/main.go` only - `pinboard` is never released and never gets ldflags. `flake.nix` sets `Version` alone, so a Nix build leaves the other three at `unknown` and `showVersion` omits the commit line. A plain `make all` leaves everything at its `-dev`/`unknown` default.

### `pinboard`

`pinboard <subcommand> [options]` - exercises the live API:
- `posts` - list, recent, add, delete, get, dates, update, suggest
- `tags` - list, rename, delete
- `user` - token, secret
- `notes` - list, get
- `version`, `help`
