# Development guide

## Prerequisites

- Go 1.24 or newer.
- Git for revision-comparison tests.
- A supported C compiler when running the Go race detector on platforms where
  it requires cgo.

No account, service, database, or external API is required.

## Core checks

```sh
go test ./...
go vet ./...
go build ./cmd/taglock
go run ./cmd/taglock check ./...
```

Run the race detector where supported:

```sh
go test -race ./...
```

Check custom vet-tool compatibility:

```sh
go build -o ./taglock-vet ./cmd/taglock
go vet -vettool="$(pwd)/taglock-vet" ./...
```

On PowerShell, resolve the executable to an absolute path before passing it to
`-vettool`.

## JSON v2

The version-gated differential test is enabled explicitly:

```sh
GOEXPERIMENT=jsonv2 go test ./semantics
```

Toolchains without the experiment should run the normal suite; only the
experiment-specific test file is excluded.

## Analyzer fixtures and fixes

Analyzer fixtures live under `analyzer/testdata/src`. Add `// want` diagnostics
for expected findings and `.golden` files for suggested edits. Tests should
include non-diagnostic cases so a rule's false-positive boundary is explicit.

When TAG metadata changes, regenerate the rule reference:

```sh
go run ./internal/rulegen
```

Review generated documentation before including it in a change.

## Fuzzing

The raw struct-tag parser accepts untrusted text and has a fuzz target:

```sh
go test -run '^$' -fuzz FuzzParse -fuzztime 30s ./tag
```

Do not commit local fuzz caches. Commit a minimized corpus entry only when it
protects against a distinct regression.

## Benchmarks

```sh
go test -run '^$' -bench . -benchmem ./tag ./contract ./rules ./baseline ./evolution
go test -run '^$' -bench . -benchmem ./snapshot ./schema ./semantics ./migration
```

Git orchestration is intentionally benchmarked separately because it creates
temporary worktrees and invokes package loading:

```sh
go test -run '^$' -bench BenchmarkGitSnapshotOrchestration -benchtime=1x ./gitdiff
```

Record the Go version, operating system, architecture, and CPU when publishing
results. See [BENCHMARKS.md](BENCHMARKS.md) for the current reference run.

## Architecture boundaries

- `tag` parses once and retains rewrite locations.
- `contract` owns normalized typed contracts and field promotion.
- `namespace` describes dialect semantics.
- `rule` contains stable TAG metadata; `rules` evaluates it.
- `semantics` models serialization implementations.
- `snapshot`, `evolution`, `schema`, `manifest`, and `verify` consume normalized
  results without reparsing source tags.
- `internal/cli` orchestrates commands without duplicating analysis logic.

Do not introduce mutable analyzer-global configuration or execute analyzed
project code from static commands.

