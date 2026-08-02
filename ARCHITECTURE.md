# Architecture

TagLock has one source-analysis core and several consumers:

```text
tag parser + namespace registry
             |
normalized contract resolver ---- go/analysis adapter + compact facts
             |
stable TAG rule engine ----------- check/fix/baseline/output
             |
serialization profiles (json/v1, json/v2)
             |
canonical snapshots
        /          \
evolution compare  schema/manifests/verification
        |
safe Git worktree orchestration
```

- `tag` parses each raw field tag once and retains lossless rewrite offsets.
- `contract` owns typed fields, suppressions, promotion paths, cycle guards,
  annotations, and resolved namespace surfaces.
- `namespace` describes dialect capabilities without a monolithic switch.
- `rule` is the stable TAG metadata catalog; `rules` evaluates contracts.
- `semantics` provides separate JSON v1/v2 models and certainty detection.
- `snapshot` canonicalizes selected exported contracts without machine paths.
- `evolution` matches stable contract identities in linear maps and classifies
  direction-aware changes separately from approvals.
- `schema`, `manifest`, and `verify` consume snapshots or engine results without
  changing analyzer behavior.
- `gitdiff` validates revisions and uses temporary detached worktrees; it never
  checks out or resets the active worktree.
- `engine` is shared by the CLI and tests. The analyzer uses the same contract
  and rule packages and exports only compact versioned facts.

The default analyzer never executes project code. Runtime test execution exists
only behind the explicit `verify run` command.
