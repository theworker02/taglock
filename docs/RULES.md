# TagLock rule reference

This file is generated from the executable rule registries.

## Source contract rules

### TAG001 — Malformed struct tag

The struct tag is not valid Go tag syntax. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `error`
- Default enabled: `true`
- Fix safety: `none`

### TAG002 — Duplicate tag namespace

A field declares the same tag namespace more than once. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `error`
- Default enabled: `true`
- Fix safety: `none`

### TAG003 — Duplicate tag option

A namespace option is repeated and has no additional effect. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `warning`
- Default enabled: `true`
- Fix safety: `safe`

### TAG004 — Unknown tag option

A tag option is not recognized by the configured namespace. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `warning`
- Default enabled: `true`
- Fix safety: `none`

### TAG005 — Empty or suspicious tag name

A tag name is empty, contains whitespace or control characters, or uses a malformed separator. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `warning`
- Default enabled: `true`
- Fix safety: `none`

### TAG101 — Ineffective tag on unexported field

Serialization tags on unexported fields are ignored by common Go encoders. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `warning`
- Default enabled: `true`
- Fix safety: `safe`

### TAG102 — Redundant explicit name

An explicit external name repeats the namespace default. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `info`
- Default enabled: `false`
- Fix safety: `safe`

### TAG103 — Ignored field has additional options

Options after an ignore marker have no effect. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `warning`
- Default enabled: `true`
- Fix safety: `safe`

### TAG104 — Duplicate external field name

Multiple direct fields resolve to the same external name. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `error`
- Default enabled: `true`
- Fix safety: `review`

### TAG105 — Promoted-field collision

Embedded or inline fields introduce an ambiguous external name. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `error`
- Default enabled: `true`
- Fix safety: `review`

### TAG106 — Suspicious case-only collision

External names differ only by case in a case-insensitive contract. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `warning`
- Default enabled: `true`
- Fix safety: `none`

### TAG201 — Option incompatible with field type

A tag option cannot operate on the field's Go type. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `error`
- Default enabled: `true`
- Fix safety: `none`

### TAG202 — Omission option has no practical effect

The selected omission behavior is ineffective for this Go type. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `warning`
- Default enabled: `true`
- Fix safety: `review`

### TAG203 — Invalid inline target

Inline behavior requires a struct, pointer-to-struct, or supported map. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `error`
- Default enabled: `true`
- Fix safety: `none`

### TAG204 — Conflicting representation options

Two options request incompatible representations. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `error`
- Default enabled: `true`
- Fix safety: `none`

### TAG301 — External naming drift

A field uses unexpectedly different external names across namespaces. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `warning`
- Default enabled: `true`
- Fix safety: `review`

### TAG302 — Inconsistent ignore policy

A field is hidden in one output namespace and exposed in another. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `warning`
- Default enabled: `true`
- Fix safety: `review`

### TAG303 — Inconsistent omission policy

A field uses materially different omission behavior across output namespaces. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `info`
- Default enabled: `true`
- Fix safety: `review`

### TAG304 — Missing required namespace

An exported field lacks a tag required by project policy. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `error`
- Default enabled: `true`
- Fix safety: `none`

### TAG305 — Inconsistent external identity

External names appear to describe different concepts rather than naming-style variants. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `warning`
- Default enabled: `true`
- Fix safety: `none`

### TAG401 — Sensitive field exposed

A security-sensitive field is publicly serialized. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `error`
- Default enabled: `true`
- Fix safety: `review`

### TAG402 — Sensitive field renamed deceptively

A sensitive field is exposed under a name unrelated to its Go identity. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `warning`
- Default enabled: `true`
- Fix safety: `none`

### TAG403 — Write-only contract mismatch

A configured write-only sensitive field is exposed by an output namespace. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `error`
- Default enabled: `false`
- Fix safety: `review`

### TAG901 — Invalid suppression

A taglock suppression is malformed or references an unknown rule. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `warning`
- Default enabled: `true`
- Fix safety: `none`

### TAG902 — Unused suppression

A valid suppression did not match any diagnostic on its declaration. TagLock evaluates the effective contract after namespace naming and field promotion rules are applied.

- Default severity: `info`
- Default enabled: `true`
- Fix safety: `none`

## Evolution rules

### EVOL001 — Public contract removed

A tracked serialized type no longer exists.

### EVOL002 — Serialized field removed

A serialized field is absent from the head contract.

### EVOL003 — Serialized field renamed

A Go field has a different external name.

### EVOL004 — Serialized field type changed

The wire representation type changed.

### EVOL005 — Optional field became required

Legacy inputs may no longer decode.

### EVOL006 — Required field became optional

Consumers may receive omitted data.

### EVOL007 — Ignored field became exposed

Previously internal data is now serialized.

### EVOL008 — Exposed field became ignored

Consumers stop receiving a field.

### EVOL009 — Embedded contract changed

Promotion path changes affect the parent surface.

### EVOL010 — Custom marshaler introduced

Static certainty is reduced by custom code.

### EVOL011 — Custom marshaler removed

Reflection-based behavior replaces custom code.

### EVOL012 — Contract certainty reduced

TagLock can no longer prove the exact wire shape.

### EVOL013 — Semantic profile changed

Snapshots use different implementation profiles.

### EVOL014 — Deprecation window violated

A field was removed without recorded deprecation.

### EVOL015 — Storage migration required

Persistent serialized data changed incompatibly.

### EVOL016 — Undocumented breaking change

No narrow approval or migration note acknowledges the break.

### EVOL901 — Invalid change approval

An approval entry is malformed or too broad.

### EVOL902 — Expired change approval

An approval is past its expiry date.

### EVOL903 — Unused change approval

An approval matched no current change.

## JSON v1-to-v2 migration rules

### JSONMIG001 — Unsupported v1 option under v2

### JSONMIG002 — New v2 interpretation

### JSONMIG003 — Field resolution changed

### JSONMIG004 — Embedded-field behavior changed

### JSONMIG005 — Omission behavior changed

### JSONMIG006 — Name matching changed

### JSONMIG007 — Custom marshaler interaction changed

### JSONMIG008 — Duplicate-name outcome changed

### JSONMIG009 — Unknown migration outcome

### JSONMIG010 — Explicit compatibility option recommended

