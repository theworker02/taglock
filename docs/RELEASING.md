# Release checklist

This checklist is for maintainers. It documents the process but does not
authorize publishing, tagging, or pushing a release.

## Before the first public release

- Choose and add an explicit software license approved by the project owner.
- Confirm module path, repository visibility, ownership, and release channels.
- Confirm security-reporting and support links are operational.

TagLock currently has no license file; do not assume a license on the project's
behalf.

## Prepare

1. Select the version and review all changes since the previous tag.
2. Update `CHANGELOG.md`, compatibility notes, examples, and installation docs.
3. Regenerate `docs/RULES.md` and inspect the diff.
4. Review dependency versions, licenses, and known security advisories.
5. Confirm no credentials, private snapshots, binaries, or local caches are
   included.

## Verify

```sh
gofmt -l .
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/taglock
go run ./cmd/taglock check ./...
GOEXPERIMENT=jsonv2 go test ./semantics
```

Also run representative snapshot, comparison, schema, SARIF, baseline, fix, and
custom vet-tool smoke tests. Review generated artifacts rather than checking
only exit status.

## Package and release

- Create an annotated semantic-version tag only after verification passes.
- Build release artifacts from the tagged source in controlled CI.
- Publish checksums and release notes describing compatibility changes.
- Test `go install github.com/magnexis/taglock/cmd/taglock@<version>` from a clean
  environment.
- Never publish automatically from an unreviewed pull request or developer
  workstation.

## After release

- Verify installation and the release page.
- Monitor regressions and security reports.
- Move completed changelog entries under the released version.
- Document any urgent workaround and prepare a patch release when required.

