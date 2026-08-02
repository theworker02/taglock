# Contributing to TagLock

Thank you for helping improve TagLock. Contributions should preserve its core
promise: diagnostics must be deterministic, evidence-based, safe to run on
untrusted source, and conservative about behavior that cannot be proven.

Please read the [code of conduct](CODE_OF_CONDUCT.md) and
[security policy](SECURITY.md) before participating.

## Before opening a change

- Search existing issues before proposing duplicate work.
- Use a security advisory rather than a public issue for vulnerabilities.
- Discuss substantial public API, snapshot-format, configuration-schema, rule,
  or compatibility-classification changes before implementation.
- Keep changes focused. Unrelated cleanup makes semantic analyzer changes much
  harder to review.

## Development setup

TagLock requires Go 1.24 or newer and Git for revision-comparison tests.

```sh
git clone https://github.com/theworker02/taglock.git
cd taglock
go test ./...
go vet ./...
go build ./cmd/taglock
go run ./cmd/taglock check ./...
```

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for JSON v2, fuzzing,
benchmarks, analyzer fixtures, and custom vet-tool instructions.

## Contribution workflow

1. Create a branch from the current default branch.
2. Add a focused regression or behavior test before changing analyzer logic.
3. Implement the smallest coherent change.
4. Run formatting, tests, vet, and TagLock's self-analysis.
5. Update user-facing documentation and the unreleased changelog when behavior
   or public APIs change.
6. Open a pull request using the repository template.

Do not include generated binaries, local caches, credentials, private source,
or production contract snapshots in a pull request.

## Adding or changing rules

A rule change must include:

- A stable identifier and metadata in the appropriate registry.
- A precise explanation of behavior, impact, and remediation.
- Positive and negative fixtures that demonstrate false-positive boundaries.
- Related source locations when multiple fields participate.
- Deterministic diagnostic ownership and ordering.
- Suggested-fix tests when the rule can fix source.
- A safety classification: `none`, `safe`, or `review`.
- Regenerated rule documentation when the TAG registry changes:

  ```sh
  go run ./internal/rulegen
  ```

Safe fixes must preserve wire behavior. Any fix that changes an external name,
exposure, omission policy, or field resolution is review-required.

## Snapshot and evolution compatibility

Snapshot and machine-output formats are public interfaces. Changes must:

- preserve canonical ordering and reproducibility;
- include stable fingerprints that do not depend only on source positions;
- reject unsupported versions rather than silently reinterpret them;
- include migration tests for a format-version change;
- classify uncertainty as unknown, never compatible;
- keep Git comparisons isolated from the active worktree.

Breaking format changes require an explicit version increment and documented
migration path.

## Tests

Choose tests according to the affected behavior:

- Unit tests for parsers, naming, configuration, fingerprints, and policies.
- `analysistest` fixtures for diagnostics, related locations, and fixes.
- Golden files for source edits and stable serialized output.
- Fixture repositories for Git behavior.
- Fuzz tests for untrusted parsers.
- Benchmarks for hot paths or changes affecting repository-scale analysis.

At minimum, run:

```sh
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go run ./cmd/taglock check ./...
```

Run `go test -race ./...` where the platform has a supported C toolchain.

## Documentation

Document only implemented behavior. Examples must use placeholder values and
must never contain credentials or private repository data. Keep commands
copyable and use current CLI names and exit behavior.

## Review expectations

Maintainers review correctness, compatibility, security, false-positive risk,
test depth, documentation, and operational cost. A change may be declined when
it cannot be made sufficiently deterministic or when it guesses at runtime
behavior TagLock cannot establish statically.

