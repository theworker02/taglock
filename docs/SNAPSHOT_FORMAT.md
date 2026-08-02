# Snapshot and comparison formats

TagLock contract snapshots use `format_version: 1`, independently of the
binary version. They record the toolchain, module identity, selected semantic
profiles, sorted packages, sorted contracts, and sorted effective fields.

Each contract contains a semantic fingerprint derived from its type identity,
contract level, profiles, certainty, custom serializer method summary, and
field surfaces. The document fingerprint covers the module and all contracts.
Source line numbers and absolute paths are excluded from fingerprints.

`--reproducible` omits `generated_at`. Canonicalization also ignores that field,
so a timestamp never changes semantic comparison results.

Field records include Go and external names, normalized type text, promotion
path, requiredness, omission behavior, nullability, ignore state, deprecation,
and validated schema annotations. No runtime values are stored.

Compatibility reports use their own `format_version: 1`. Each change has a
stable EVOL identifier and fingerprint, profile, direction, severity, before
and after values, explanation, recommendation, and optional approval metadata.
Confirmed renames also include `rename_confidence`. Release-history policy is
configuration input and is not embedded into the serialized contract itself;
the compared module versions remain part of the snapshot metadata.

Snapshot readers reject unknown top-level fields and unsupported versions. A
future format migration must be explicit rather than silently reinterpreting
old data.
