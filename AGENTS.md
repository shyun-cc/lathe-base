# Repository guidelines

- Keep `docs/` for the public user manual. Authorized maintainers keep internal requirements, architecture and decisions in the private Gitea repository `shw-documents/lathe-base`.
- Build for one factory per deployment. Keep sales, production, outsourcing, shipping, settlement, invoice registration, and printing in their domain packages under `apps/api/internal/module`.
- Use the public GoFrame and shw-gf-lib dependencies directly. Use Vue 3, TypeScript, and the public shw-wui project for the responsive Web client.
- Keep general workflow, document identity, quantity limits, and financial invariants in the core. Add industry-specific data and behavior through module-owned contracts under `extensions/`.
- Use integer thousandths for quantities and integer minor units for money. Enforce cross-row limits inside transactions. Persist reconciliation allocations separately from cash entries and preserve the party, currency, and direction match.
- Keep migrations schema-only and immutable after release. Add a numbered Goose SQL file per table or other schema object. Use the explicit migration command and verify against a fresh PostgreSQL 18 database.
- Commit public, synthetic examples only. Keep customer data, credentials, private domains, proxies, and internal deployment files outside the repository.
- Keep all three READMEs aligned. Before pushing, check `gofmt`, `go vet ./...`, `go test ./...`, and run `gitleaks git --redact`.
