## Summary

Describe the problem and the implemented outcome.

## Compatibility and safety

- Affected rule or command:
- Public API, configuration, snapshot, or output-format impact:
- False-positive and uncertainty considerations:
- Fix safety classification, if applicable:

## Verification

List the exact commands run and their results.

- [ ] Changed Go files are formatted.
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] `go run ./cmd/taglock check ./...` passes.
- [ ] Race, fuzz, benchmark, JSON v2, or vet-tool checks were run when relevant.
- [ ] Tests cover the successful case and false-positive boundary.
- [ ] Machine-readable output remains deterministic where affected.
- [ ] Documentation and `CHANGELOG.md` are updated when behavior changed.
- [ ] No secrets, private snapshots, generated binaries, or unrelated changes are included.

## Review notes

Call out generated files, review-required fixes, known limitations, or follow-up
work that should not block this change.

