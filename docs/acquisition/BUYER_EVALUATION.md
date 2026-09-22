# Buyer evaluation â€” TagLock

## Goal

In 15â€“45 minutes, verify the Product builds or runs as documented and that proprietary notices are present.

## Steps

1. Confirm root `LICENSE` is proprietary and `ACQUISITION.md` exists.
2. Skim `README.md` install/run claims.
3. Execute:

```
```sh
go install github.com/theworker02/taglock/cmd/taglock@latest
```
```sh
taglock check ./...
taglock check --format json --fail-on error ./...
taglock check --format sarif ./... > taglock.sarif
taglock check --format github ./...
taglock check --json-semantics v2 ./...
```
```sh
taglock fix ./...
taglock fix --rule TAG003 ./...
taglock fix --diff ./...
taglock fix --review ./... # explicitly include wire-changing suggestions
```
```go
type LegacyPayload struct {
    //taglock:ignore TAG301 -- public API retains its legacy YAML name
    Name string `json:"display_name" yaml:"displayName"`
}
```
```sh
taglock snapshot ./...
taglock snapshot --semantics both --reproducible --output .taglock/contracts.json ./...
```
```sh
taglock compare --format markdown old.json new.json
```
```sh
taglock compare --base main --head HEAD --format markdown ./...
taglock compare --base v1.4.0 --head v1.5.0 --format sarif --output taglock.sarif ./...
```
```yaml
evolution:
  require_deprecation_before_removal: true
  minimum_deprecation_releases: 2
  release_history: [1.4.0, 1.5.0, 2.0.0]
```
```sh
```

4. Run tests if present (`npm test`, `pytest`, `cargo test`, `go test ./...`, etc.).
5. Record README vs observed behavior gaps in workpapers.

## Pass criteria

- [ ] Clone succeeds
- [ ] Documented happy path works **or** failure is explained
- [ ] Minimal path needs no surprise secrets
- [ ] License notices intact

*Updated: 2026-09-22*
