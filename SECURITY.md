# Security policy

TagLock analyzes source and build metadata that may be untrusted. Security
reports are taken seriously, especially when they involve unintended code
execution, filesystem changes, worktree corruption, secret disclosure,
diagnostic suppression bypasses, unsafe automatic fixes, or denial of service.

## Supported versions

Before the first tagged release, security fixes are applied to the default
branch. After releases begin, the latest release line and the default branch
will receive security fixes. Older versions may be asked to upgrade when a safe
backport is impractical.

| Version | Supported |
| --- | --- |
| Default branch | Yes |
| Latest tagged release | Yes, once published |
| Older releases | Best effort |

## Reporting a vulnerability

Use [GitHub private vulnerability reporting](https://github.com/magnexis/taglock/security/advisories/new)
when available. If it is unavailable, contact the repository owner privately.
Do not open a public issue before coordinated disclosure.

Include:

- The affected command, package, and revision.
- Impact and the attacker-controlled input.
- A minimal reproducer without private source or credentials.
- Required Go, Git, operating-system, and configuration conditions.
- Any known workaround.

Never include real tokens, credentials, production data, or private repository
contents. Maintainers will acknowledge reports as capacity permits, validate
impact, coordinate a fix and advisory, and credit reporters who request credit.
As an independently maintained project, TagLock does not promise a fixed
response or remediation time.

## Trust model

- Static check, snapshot, comparison, migration, schema, and manifest engines
  never load project binaries as runtime plugins.
- Package loading invokes the installed Go toolchain and inherits normal Go
  environment behavior, including `GOWORK`, `GOFLAGS`, module proxies, vendor
  configuration, and private-module settings.
- Git comparison invokes `git` with argument arrays rather than a shell and
  analyzes locally available revisions in isolated temporary worktrees.
- TagLock never fetches Git remotes automatically. Missing revisions must be
  fetched through the operator's normal authenticated workflow.
- `taglock verify generate` is an explicit source-writing command. Only
  `taglock verify run` executes generated project tests.
- Suggested fixes use parser-derived source spans. Safe fixes must preserve wire
  behavior; wire-changing suggestions require explicit review selection.
- Configuration and annotations are treated as data, not executable input.
- Snapshots contain type and contract metadata, never runtime field values.
- Sensitive-field diagnostics are heuristics and do not guarantee that a model
  is safe to expose.

Run TagLock with the same dependency, network, filesystem, and process isolation
used for other Go build tools in CI. Analyzed packages can still influence Go
toolchain work through ordinary build metadata and dependency resolution.

## Security boundaries

The following are not automatically vulnerabilities, though reports of unsafe
behavior around them are welcome:

- A false negative caused by an arbitrary custom marshaler that TagLock marks
  as opaque.
- Resource consumption proportional to a deliberately enormous valid package,
  unless it is unexpectedly unbounded or enables practical denial of service.
- Diagnostics from unsupported third-party tag semantics.
- Project code executed only after an operator explicitly runs generated tests.

## Disclosure and releases

Security fixes should include a regression test when safe, an impact statement,
affected versions, upgrade guidance, and credit preferences. Release artifacts
must not be published until tests, vet, self-analysis, dependency review, and
secret checks complete.

