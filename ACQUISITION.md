# Acquisition Brief â€” TagLock

**Date:** 2026-09-22  
**Repository:** https://github.com/theworker02/taglock  
**Default branch:** `main`  
**Primary language:** Go  
**Status:** Diligence briefing only. **No acquisition has occurred** by virtue of this file.  
**License:** Proprietary â€” sale, written commercial license, or completed asset transfer required (see root `LICENSE`).  
**Valuation:** Not stated.  
**Contact:** GitHub [@theworker02](https://github.com/theworker02) Â· [thanks.dev/u/gh/theworker02](https://thanks.dev/u/gh/theworker02)

> Cloning or forking this repository does **not** grant production, redistribution, SaaS, OEM, or commercial rights.

---

## 1. Executive thesis

<img src="docs/assets/taglock-logo.png" alt="TagLock logo: a padlock containing code brackets" width="180"> <a href="https://theworker02.github.io/taglock/">Website</a> Ã‚Â· <a href="CONTRIBUTING.md">Contributing</a> Ã‚Â·

**Why a buyer cares:** TagLock packages transferable product IP â€” source, docs, in-repo brand assets, and a diligence room under `docs/acquisition/` â€” under a clear proprietary posture so diligence can proceed without mistaking the repo for open source.

---

## 2. Product snapshot

| Item | Detail |
|------|--------|
| Product | TagLock |
| Repo | `theworker02/taglock` |
| Language | Go |
| Open source? | **No** â€” proprietary |
| Rightsholder | theworker02 |
| Diligence pack | `docs/acquisition/` |

### Capability highlights (from current materials)

- [Architecture](ARCHITECTURE.md)
- [Contribution guide](CONTRIBUTING.md)
- [Development guide](docs/DEVELOPMENT.md)
- [Compatibility policy](docs/COMPATIBILITY.md)
- [Security policy](SECURITY.md)
- [Support](SUPPORT.md)
- [Code of conduct](CODE_OF_CONDUCT.md)
- [Governance](GOVERNANCE.md)
- [Release checklist](docs/RELEASING.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [GitHub Pages setup](docs/GITHUB_PAGES.md)

---

## 3. Problem / opportunity

Teams evaluating TagLock typically need either (a) a commercial right to run or embed it, or (b) outright ownership of the Product IP for strategic build-out. Public GitHub visibility without a proprietary license creates false assumptions about free production use. This brief and the linked data room make the commercial path explicit.

---

## 4. What ships today

Honest maturity: treat repository contents, README claims, tests, and release tags as the source of truth. Do not assume production customers, ARR, filed patents, or SLAs unless separately evidenced in diligence.

Typical transferable surfaces:

- Source tree and build/test scripts present in-repo
- Documentation and design notes
- Acquisition / diligence markdown under `docs/acquisition/`
- Branding assets committed to the repository (if any)

---

## 5. Demo / evaluation path (buyer)

Minimal path (no secrets required unless README says otherwise):

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

Extended evaluation: `docs/acquisition/BUYER_EVALUATION.md`. Written NDA / evaluation grants may be required for private materials.

---

## 6. What a transaction typically includes

Subject to definitive schedules:

| Included (typical) | Excluded (typical) |
|--------------------|--------------------|
| Repo materials + asserted original IP | Seller personal accounts / unrelated repos |
| Docs + diligence room at closing | Third-party dependency source under separate licenses |
| In-repo brand marks as assigned | Secrets without rotation plan |
| Know-how captured in docs | Fabricated revenue, user, or adoption metrics |

---

## 7. Suggested deal structures

| Structure | When it fits |
|-----------|--------------|
| Non-exclusive commercial license | Deploy/run under seat or environment terms |
| Exclusive field-of-use license | Buyer wants exclusivity; seller may retain entity |
| Asset / IP assignment | Buyer wants ownership of Materials outright |
| OEM / redistribution | Separate agreement â€” not implied here |

Commercial terms (price, earnouts, escrow) are negotiated under NDA with counsel.

---

## 8. Buyer diligence checklist

- [ ] Confirm Rightsholder identity and authority to sell/license
- [ ] Inventory Materials (`docs/acquisition/ASSET_INVENTORY.md`)
- [ ] Review IP posture (`IP_PROVENANCE.md`) and dependencies (`DEPENDENCY_INVENTORY.md`)
- [ ] Run evaluation script (`BUYER_EVALUATION.md`)
- [ ] Review risks (`RISK_REGISTER.md`)
- [ ] Agree transfer scope (`TRANSFER_MANIFEST.md`) and handoff (`HANDOFF_CHECKLIST.md`)
- [ ] Supersede root `LICENSE` at closing via definitive agreement

---

## 9. Related documents

| Document | Purpose |
|----------|---------|
| `LICENSE` | Proprietary â€” no default grant |
| `docs/acquisition/README.md` | Data-room index |
| `docs/acquisition/EXECUTIVE_SUMMARY.md` | One-page thesis |
| `README.md` | Product overview |
| `SECURITY.md` | Vulnerability reporting |
| `COMMERCIAL.md` | Licensing contact path |
| `.github/FUNDING.yml` | Sponsors / thanks.dev |

---

## 10. Disclaimer

This package is informational and **does not** create a binding offer, grant of rights, or investment advice. Engage counsel for any transaction.

---

*Document version: 2.0.0 / 2026-09-22 Â· Classification: acquisition briefing*
