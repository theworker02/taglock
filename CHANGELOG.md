# Changelog

All notable changes to TagLock will be documented here.

## Unreleased

No unreleased changes yet.

## 0.1.0 - 2026-08-02

### Added

- Original TagLock logo and a responsive, dependency-free GitHub Pages site.
- Official Pages deployment workflow using a least-privilege deployment token.
- Contribution, support, conduct, governance, development, compatibility,
  release, troubleshooting, and Pages documentation.
- Structured GitHub bug, feature, and pull-request templates.
- Release-history-backed deprecation windows and explicit replacement rename
  confirmation.
- Validated generic instantiations for generated runtime verification tests.
- Go 1.24-1.26 CI coverage and an experimental JSON v2 differential test job.

- Reusable `go/analysis` analyzer and standalone `taglock` command.
- Strict YAML configuration discovery and explicit `-config` support.
- Semantic rules for duplicate names, embedded collisions, tag options,
  validation conflicts, naming drift, sensitive exposure, database names,
  omission behavior, field visibility, and field-type compatibility.
- Safe suggested fixes for configured naming conventions.
- Unit and analyzer integration tests.
- Separate JSON v1 and version-aware JSON v2 semantic profiles.
- Canonical contract snapshots, compact manifests, and stable fingerprints.
- Direction-aware EVOL comparison with deprecation and narrow approvals.
- JSON v1-to-v2 migration reports using JSONMIG001–JSONMIG010.
- JSON Schema and OpenAPI component generation with drift checking.
- Safe isolated Git revision comparison and Markdown/JSON/SARIF reports.
- Explicit runtime verification-test generation and analyzer contract facts.
