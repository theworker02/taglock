# TagLock triage

The `triage` package converts analyzer-independent `rules.Diagnostic` values into deterministic summaries suitable for CI annotations, dashboards, pull-request bots, and security reporting.

```go
summary := triage.Summarize(diagnostics)

fmt.Println(summary.Score)
fmt.Println(summary.BySeverity)
fmt.Println(summary.TopRules(5))

if summary.FailsAt(rule.SeverityError) {
    os.Exit(1)
}
```

The summary includes:

- total findings and weighted risk score;
- counts by severity, rule, category, and tag namespace;
- total fixable findings;
- safe-fix and review-required fix counts;
- deterministic top-rule ranking;
- threshold evaluation using TagLock's public severity type.

`Summarize` does not mutate, reorder, suppress, or reinterpret the original diagnostics.
