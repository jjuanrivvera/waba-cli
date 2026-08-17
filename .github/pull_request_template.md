## What this changes

## Why

## Checklist

- [ ] `make verify` passes (fmt, vet, lint, tests, spec-check, spec-completeness, coverage ≥80%, DoD)
- [ ] New commands ship with tests in this commit
- [ ] Surface changes update `tools/genspec/resources.json` and regenerate `api-manifest.json` (`make spec-gen`)
- [ ] Docs regenerated (`make docs-gen`) if the command tree changed
- [ ] Any ambiguous API assumption is recorded in `DECISIONS.md`
