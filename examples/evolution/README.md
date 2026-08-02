# Contract evolution example

These build-tagged source variants illustrate a stable API, a compatible
optional-field addition, and breaking changes involving names, types,
embedding, sensitive exposure, and a custom marshaler.

The v1 contract also records a field deprecation. The breaking variant removes
that field, uses JSON v2 `inline` behavior, changes the ID wire type, exposes a
sensitive field, and introduces an opaque custom marshaler. `changes.yml` shows
the narrow approval shape for an intentional removal, while `schemas/` contains
the JSON Schema form of the original public contract.

Analyze one variant by copying it into a small module or enabling the
`taglock_evolution_example` build tag. The directories intentionally use
different package paths, so the tutorial demonstrates snapshot comparison with
the same package identity instead of comparing these directories directly.
