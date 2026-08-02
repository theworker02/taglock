# Support

TagLock is an independent open-source developer tool. Support is provided on a
best-effort basis and no response-time guarantee is implied.

## Where to ask

- Bugs and reproducible analyzer failures: use the GitHub bug-report form.
- Feature and rule proposals: use the feature-request form.
- Vulnerabilities: follow [SECURITY.md](SECURITY.md); do not file a public issue.
- General usage questions: open a focused issue with the command, Go version,
  relevant configuration, and a minimal source example.

Before reporting a problem, run:

```sh
go version
taglock config validate
taglock check --format text ./...
```

Include the complete error, expected behavior, package pattern, operating
system, and whether a workspace or vendor directory is in use. Redact module
proxy credentials, private import paths when necessary, source values, and all
secrets.

## Supported scope

Support covers released TagLock behavior, the current documented configuration
schema, supported Go versions, analyzer embedding, and the standalone CLI.
Help with private build infrastructure, custom serializers, third-party
validator semantics, or unreleased Go experiments may require a minimal public
reproducer.

