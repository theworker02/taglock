# Security

TagLock is a static analyzer. Its check, snapshot, compare, migration, schema,
and manifest engines do not execute the program being analyzed, read
application secrets, or send source code over the network. Runtime verification
is a separate, explicit command.

## Trust model

- Treat analyzed repositories and their build metadata as untrusted input.
- Package loading invokes the installed Go toolchain. Git-revision comparison
  invokes the installed `git` executable with argument arrays (never through a
  shell) and analyzes revisions in isolated temporary worktrees.
- `taglock verify generate` writes clearly marked test files. Only
  `taglock verify run` executes those project tests.
- Configuration is strict and bounded to known namespaces and naming styles.
- Suggested fixes replace only a struct-tag literal at a source span supplied
  by the Go parser. Competing whole-tag edits are suppressed.
- Sensitive-field diagnostics are heuristics. They are a review signal, not a
  guarantee that a model is safe to expose.

Go package loading may evaluate normal Go toolchain metadata and can be affected
by the caller's standard Go environment (`GOWORK`, `GOFLAGS`, module proxies,
and private-module settings). Run TagLock with the same isolation and dependency
policy used for other Go build tools in CI.

Git comparison requires a repository and locally available revisions. TagLock
does not fetch remotes, alter branches, reset the caller's worktree, or include
uncommitted changes unless they have first been represented by a revision.
Missing-revision errors explicitly direct the operator to fetch through their
normal authenticated Git workflow; TagLock never initiates that network action.

## Reporting vulnerabilities

Please report suspected vulnerabilities privately to the repository owner. Do
not include real credentials, private source code, or production data in a
report. Include a minimal reproducer, affected version, and expected impact.
