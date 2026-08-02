<p align="center">
  <img src="docs/assets/taglock-logo.png" alt="TagLock logo: a padlock containing code brackets" width="180">
</p>

# TagLock

<p align="center">
  <a href="https://magnexis.github.io/taglock/">Website</a> ·
  <a href="docs/RULES.md">Rules</a> ·
  <a href="CONTRIBUTING.md">Contributing</a> ·
  <a href="SECURITY.md">Security</a>
</p>

**Compile-time confidence for Go's runtime metadata — and compatibility
intelligence for the contracts that metadata creates.**

TagLock statically analyzes Go struct tags as one serialized contract. It finds
today's collisions, unsafe exposure, type errors, and namespace drift, then can
snapshot those external representations and detect changes that may break APIs,
stored JSON, configuration, or downstream consumers tomorrow.

TagLock never executes analyzed project code during static analysis, snapshot,
schema, or comparison commands.

## Install

TagLock requires Go 1.24 or newer.

```sh
go install github.com/magnexis/taglock/cmd/taglock@latest
```

## Source-contract analysis

```sh
taglock check ./...
taglock check --format json --fail-on error ./...
taglock check --format sarif ./... > taglock.sarif
taglock check --json-semantics v2 ./...
```

Diagnostics have stable identifiers such as `TAG104` (duplicate external
field), `TAG301` (naming drift), and `TAG401` (sensitive exposure). Text output
uses ordinary Go source locations; JSON and SARIF include related collision
locations, fix metadata, and semantic-profile properties.

Safe fixes are the default:

```sh
taglock fix ./...
taglock fix --rule TAG003 ./...
taglock fix --diff ./...
taglock fix --review ./... # explicitly include wire-changing suggestions
```

Review-required changes are never silently applied.

Suppress a precise finding on the next declaration:

```go
type LegacyPayload struct {
    //taglock:ignore TAG301 -- public API retains its legacy YAML name
    Name string `json:"display_name" yaml:"displayName"`
}
```

Malformed and unused suppressions are reported as `TAG901` and `TAG902`.

## Contract snapshots

A serialized contract is the externally observable field surface after tag
naming, omission, ignore, embedding, promotion, and implementation-specific
resolution rules have been applied.

```sh
taglock snapshot ./...
taglock snapshot --semantics both --reproducible --output .taglock/contracts.json ./...
```

Snapshots select exported contracts by default and can use
`//taglock:contract public`, `storage`, `message`, `configuration`, `internal`,
or `experimental` annotations. Output is canonical, path-normalized, sorted,
fingerprinted, and free of runtime field values or secrets.

## Compare evolution

Compare checked-in snapshots:

```sh
taglock compare --format markdown old.json new.json
```

Or compare Git revisions in isolated temporary worktrees:

```sh
taglock compare --base main --head HEAD --format markdown ./...
taglock compare --base v1.4.0 --head v1.5.0 --format sarif --output taglock.sarif ./...
```

The current worktree and uncommitted changes are not checked out, reset, or
modified. Evolution findings distinguish producer, consumer, bidirectional,
storage, configuration, message, and advisory compatibility. Stable `EVOL`
rules cover removal, rename, type and requiredness changes, exposure, embedding,
custom marshalers, certainty reduction, deprecation, and storage migration.

Intentional breaks can be acknowledged narrowly in `.taglock/changes.yml`.
Approvals remain visible in machine output and require a reason and migration
note. Wildcard, expired, invalid, and unused approvals are rejected or reported.

Deprecation windows can be proven from explicit project release history:

```yaml
evolution:
  require_deprecation_before_removal: true
  minimum_deprecation_releases: 2
  release_history: [1.4.0, 1.5.0, 2.0.0]
```

The history must be unique and strictly increasing. Replacement annotations can
confirm a rename across changed Go field names; reports expose
`rename_confidence: confirmed`. TagLock does not promote weak similarity into a
definitive rename.

## JSON v1 and v2

`json/v1` and `json/v2` are separate semantic profiles. TagLock records the Go
toolchain and whether the experimental v2 package is active. Custom
`MarshalJSON`/`UnmarshalJSON` methods reduce certainty to `opaque`; text
marshalers reduce it to `partial`. TagLock never claims to know arbitrary custom
wire behavior.

```sh
taglock migrate json-v2 ./...
taglock migrate json-v2 --format json ./...
```

Migration findings use `JSONMIG001`–`JSONMIG010` for option, field resolution,
embedding, omission, name matching, custom method, ambiguity, and explicit
compatibility differences. On toolchains where `encoding/json/v2` is present
but `GOEXPERIMENT=jsonv2` is inactive, TagLock can model the known profile but
reports that runtime availability distinction.

## Schema and manifests

```sh
taglock schema --format json-schema ./...
taglock schema --format openapi --output openapi-components.json ./api/...
taglock schema check ./...
taglock manifest module --output taglock.manifest.json ./...
```

Schema export supports primitives, aliases through their underlying wire type,
pointers/nullability, slices, arrays, maps, nested references, recursion,
`time.Time`, byte slices, required/optional fields, and schema annotations.
Opaque custom marshalers produce explicit `x-taglock-certainty` metadata rather
than a guessed schema. OpenAPI output contains component schemas only; TagLock
does not claim to understand routes or generate a complete OpenAPI document.

## Explicit runtime verification

```sh
taglock verify generate ./api/...
taglock verify run ./api/...
```

Generation writes clearly marked Go tests using zero values and configured
fixtures. Only `verify run` executes project tests; the normal analyzer remains
fully static.

Generic declarations are generated only when the project supplies a valid
package-scope instantiation:

```yaml
verification:
  fixtures:
    example/api.Page:
      file: testdata/contracts/page.json
      type_arguments: [string]
```

Type arguments are parsed and checked against the declaration's constraints
before any test file is written. Configured fixtures are also decoded,
re-encoded, and compared as normalized JSON, providing an explicit runtime
oracle for custom marshalers that static analysis correctly treats as opaque.

## Configuration and adoption

```sh
taglock init
taglock config validate
taglock config print
taglock baseline create ./...
taglock check --baseline .taglock-baseline.json ./...
```

TagLock discovers `.taglock.yml`, `.taglock.yaml`, `taglock.yml`, or
`taglock.yaml` by walking upward. The versioned schema validates unknown fields,
rules, severities, semantic profiles, compatibility modes, contract levels, and
glob syntax. Legacy Phase 1 list-style configuration remains readable.

See [.taglock.example.yaml](.taglock.example.yaml), the generated
[rule reference](docs/RULES.md), the [snapshot format](docs/SNAPSHOT_FORMAT.md),
and the [JSON API evolution tutorial](docs/tutorials/protecting-a-json-api.md).

## Embed TagLock

```go
package main

import (
    "github.com/magnexis/taglock/analyzer"
    "golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(analyzer.Analyzer) }
```

`analyzer.New(config.Config)` remains available for Phase 1 compatibility.
`analyzer.NewWithOptions` provides validated in-memory or fixed-path policies.
The analyzer exports compact, versioned contract facts for exported types and
is compatible with `go vet -vettool` invocation.

## Development

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/taglock
go run ./cmd/taglock check ./...
go test -bench . ./...
```

## Project documentation

- [Architecture](ARCHITECTURE.md)
- [Contribution guide](CONTRIBUTING.md)
- [Development guide](docs/DEVELOPMENT.md)
- [Compatibility policy](docs/COMPATIBILITY.md)
- [Security policy](SECURITY.md)
- [Support](SUPPORT.md)
- [Code of conduct](CODE_OF_CONDUCT.md)
- [Governance](GOVERNANCE.md)
- [Release checklist](docs/RELEASING.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [GitHub Pages setup](docs/GITHUB_PAGES.md)

Known limitations are documented in [SECURITY.md](SECURITY.md) and
[docs/SNAPSHOT_FORMAT.md](docs/SNAPSHOT_FORMAT.md). Reproducible performance
measurements are documented in [docs/BENCHMARKS.md](docs/BENCHMARKS.md). JSON v2
remains experimental and arbitrary custom serializer behavior cannot be proven
statically; CI runs the version-gated differential suite with
`GOEXPERIMENT=jsonv2` on Go 1.26.
