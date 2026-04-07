# Review Change

Review a change in this repository.

Priorities:

1. Behavioral regressions in the changed request, worker, or startup path
2. Broken auth or workspace isolation
3. Incorrect layering between router, handler, service, and persistence code
4. Missing migration, validation, or sync-backend coverage
5. Documentation drift

Focus questions:

- Does the change belong in the package where it was implemented?
- Does it preserve auth, tenant, and workspace constraints?
- Are request validation, failure handling, and default values safe?
- If persistence changed, are migrations and GORM semantics correct?
- If sync behavior changed, do memory and Redis paths remain aligned?
- Does the implementation still match the relevant guidance in `README.md` or `./docs`?

Output format:

- findings first, ordered by severity
- open questions or assumptions
- short summary only after findings
