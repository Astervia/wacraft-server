# Docs Map

Use this file to navigate the broader repository documentation under `./docs`.
The `.agent/` folder should stay lightweight and refer to these docs instead of
duplicating them.

## Start Here

- `README.md`
  - development workflow
  - Docker commands
  - migration commands
  - Swagger generation
- `docs/features/`
  - feature-specific implementation context
- `docs/guides/`
  - operator and testing guidance

## Architecture and Cross-Cutting Features

- `docs/features/multi_tenant/MULTI_TENANT_IMPLEMENTATION_PLAN.md`
  - tenant and workspace model
- `docs/features/multi_tenant/MULTI_TENANT_IMPLEMENTATION_SUMMARY.md`
  - current multitenancy implementation state
- `docs/features/horizontal_scaling/HORIZONTAL_SCALING_IMPLEMENTATION_PLAN.md`
  - distributed deployment strategy
- `docs/features/horizontal_scaling/HORIZONTAL_SCALING_REQUIREMENTS.md`
  - scaling assumptions and constraints
- `docs/features/horizontal_scaling/HORIZONTAL_SCALING_TEST_PLAN.md`
  - test scope for multi-instance behavior
- `docs/features/webhook/WEBHOOK_SYSTEM_IMPLEMENTATION_SUMMARY.md`
  - webhook ingestion and delivery architecture

## Feature Areas

- `docs/features/auth/README.md`
  - auth flows and security-related behavior
- `docs/features/billing/`
  - subscription, throughput, pricing, and webhook billing behavior
- `docs/features/firewall/FIREWALL_IMPLEMENTATION_SUMMARY.md`
  - IP filtering and firewall-related behavior
- `docs/features/campaign_scheduling/campaign_scheduling.md`
  - campaign scheduling design

## Operator Guides

- `docs/guides/billing/STRIPE_SETUP_GUIDE.md`
  - Stripe environment and webhook setup
- `docs/guides/billing/BILLING_RATE_LIMIT_DETECTION.md`
  - throughput behavior and troubleshooting
- `docs/guides/campaign_scheduling/campaign_scheduling_frontend.md`
  - frontend integration expectations for scheduling
- `docs/guides/campaign_scheduling/campaign_scheduling_ui_testing.md`
  - manual testing guidance for scheduling flows

## Generated API Docs

- `docs/swagger.yaml`
  - generated OpenAPI spec
- `docs/swagger.json`
  - generated OpenAPI spec in JSON

## Usage Guidance

- For implementation tasks, read the nearest feature doc before editing.
- For review tasks, compare code against the relevant feature summary or guide and call out drift.
- If a topic is already covered in `./docs`, prefer pointing to it over rewriting the same background in `.agent/`.
