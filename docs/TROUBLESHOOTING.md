# Troubleshooting

## Analysis cannot load packages

Run the equivalent Go command from the same directory:

```sh
go list ./...
go env GOWORK GOFLAGS GOPROXY GOMODCACHE GOCACHE
```

Resolve missing dependencies, workspace configuration, build tags, or cache
permissions first. TagLock does not silently skip packages that fail loading.

## Configuration is not found

TagLock searches upward from the working directory for `.taglock.yml`,
`.taglock.yaml`, `taglock.yml`, or `taglock.yaml`. Use `--config` to select a
different file and run:

```sh
taglock config validate --config path/to/taglock.yaml
taglock config print --config path/to/taglock.yaml
```

## A Git revision is unavailable

Git comparison uses only local objects. Fetch the exact branch, tag, or commit
through the repository's normal authenticated Git workflow, then retry. TagLock
does not fetch automatically or alter the active branch.

## JSON v2 is reported unavailable

Static migration modeling can still run. Actual v2 differential tests require a
toolchain containing `encoding/json/v2` and the appropriate experiment:

```sh
GOEXPERIMENT=jsonv2 go test ./semantics
```

## A custom marshaler is opaque

This is intentional: arbitrary Go methods cannot be modeled safely. Configure a
runtime fixture and run `taglock verify generate`, review the generated test,
then explicitly run `taglock verify run`.

## Race tests fail before package compilation on Windows

The Go race detector may require a working 64-bit C compiler through cgo. Check
`go env CC CGO_ENABLED GOARCH` and repair the compiler installation. Do not
disable the race detector and report success as equivalent verification.

## A suggested fix is not applied

Safe fixes are the default. Wire-changing edits are review-required and need an
explicit review opt-in. TagLock also suppresses overlapping edits rather than
risk corrupting a struct-tag literal.

