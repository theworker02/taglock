# Compatibility policy

TagLock has several public interfaces. Changes should identify which interface
they affect and preserve its stability guarantees.

## Go API

The supported embedding surface is:

```go
analyzer.Analyzer
analyzer.New(config.Config)
analyzer.NewWithOptions(...)
```

Internal packages may evolve, but public package changes should remain source
compatible unless a documented major-version migration is necessary.

## Rule identifiers

Published `TAG`, `EVOL`, and `JSONMIG` identifiers are stable identities.
Messages and remediation may become clearer, but an identifier must not be
silently reassigned to unrelated behavior.

## Configuration

Configuration is versioned independently through its `version` field. Unknown
fields and unsupported values fail with actionable errors. A breaking schema
change requires a new schema version and documented migration.

## Snapshot and report formats

Snapshots and compatibility reports contain explicit format versions.
Canonical ordering and fingerprint identity are part of the machine-readable
contract. Readers reject unsupported versions instead of guessing.

Source line numbers are not durable identities. Fingerprints instead use rule,
package, contract, field path, namespace or profile, and normalized diagnostic
or change identity.

## CLI behavior

Command names, documented flags, output schemas, and exit codes are public:

- `0`: no finding at or above the configured threshold.
- `1`: policy or compatibility findings.
- `2`: invalid CLI input or configuration.
- `3`: analysis could not complete.

Human-readable wording may improve. Scripts should consume JSON or SARIF rather
than parse text output.

## Serialization certainty

TagLock never treats reduced certainty as compatibility. Custom behavior is
reported as partial or opaque and may require explicit runtime fixtures.

Tagged releases are intended to follow semantic versioning once publishing
begins. Until then, the default branch is the authoritative development line.

