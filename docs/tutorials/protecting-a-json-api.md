# Protecting a JSON API with TagLock

1. Define and annotate a public contract:

   ```go
   //taglock:contract public
   type User struct {
       ID string `json:"id"`
       Email string `json:"email,omitempty"`
   }
   ```

2. Capture the initial contract:

   ```sh
   taglock snapshot --semantics both --output contracts-v1.json ./api/...
   ```

3. Add an optional `AvatarURL` field and create `contracts-compatible.json`.
   Comparing the snapshots reports a compatible optional-field addition.

4. Rename `email` to `primary_email`, create another snapshot, and compare:

   ```sh
   taglock compare --format markdown contracts-v1.json contracts-breaking.json
   ```

   `EVOL003` explains that both producers and consumers may break.

5. Before removal, deprecate the old field:

   ```go
   // Deprecated: use PrimaryEmail.
   //taglock:deprecated since=v1.4.0 remove-after=v2.0.0 replacement=PrimaryEmail
   Email string `json:"email,omitempty"`
   ```

6. Generate a schema and check it into source control:

   ```sh
   taglock schema --format json-schema --output schemas/contracts.json ./api/...
   taglock schema check ./api/...
   ```

7. Inspect v2 migration behavior:

   ```sh
   taglock migrate json-v2 ./api/...
   ```

8. Add `.github/workflows/taglock-contracts.yml` using the reusable example in
   `docs/examples/github-actions-contracts.yml`. Fetch full Git history so base
   and head revisions can be isolated safely.

9. If a major-version break is intentional, add a narrow approval containing
   the exact EVOL rule, contract, field, reason, and migration note. Validate it
   with `taglock changes validate`; the acknowledged diagnostic remains in JSON,
   Markdown, and SARIF reports.
