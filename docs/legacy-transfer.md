# Legacy transfer and verification

This is the engineering work record for transferring behavior from
`https://github.com/etoosammy01/Agro-Shield.git` into the architecture in
[`README.md`](../README.md). File existence and successful compilation do not
establish that a workflow works.

## Comparison baseline

- Upgrade baseline: `e5f07b4`.
- Legacy baseline: `1c353807d0ad6daa4964c1b8ede8908b10a4a0b6`.
- The legacy repository was cloned separately for inspection. Its database,
  credentials, generated archives, and application entry point are not inputs
  to the upgraded application's runtime.
- Equivalent legacy directories (`handlers`, `routes`, `middleware`, `render`,
  `ussd`) map to `backend/internal/`. Legacy farmers and crops map to users and
  products. Existing upgraded delivery, review, and location domains remain the
  reference model.

## Execution plan

1. **Baseline and comparison:** run existing checks, inventory transferred and
   missing behavior, inspect both database models and all public routes.
2. **Foundation:** complete configuration wiring, authentication and password
   recovery, authorization, session lifecycle, HTTP protections, structured
   locations, and onboarding. External services must report an unavailable
   configuration honestly.
3. **Marketplace and money:** verify products, cart, negotiation, orders,
   payment callbacks/webhooks, wallet preservation, delivery, and reviews using
   PostgreSQL. Protect ownership, inventory, duplicate requests, and amounts.
4. **Learning and communication:** connect the learning experience to stored
   journeys and a configurable AI provider, enforce ownership, derive progress
   from stored checkpoints, and verify chat and notifications.
5. **Real data and operations:** replace runtime in-memory market persistence,
   restore compatible import capabilities, and remove fabricated results and
   disconnected controls. Business settings belong in validated configuration;
   domain states and explicit defaults remain named constants.
6. **Verification and handoff:** exercise fresh migrations, repeat migrations,
   legacy archive/backfill, service and HTTP workflows, templates and browser
   assets; run the Go tests, race detector, vet, and build. Record evidence and
   any checks requiring external credentials.

## Design boundaries

The agreed target is Go `net/http`, PostgreSQL, and HTML/CSS/JavaScript with
`handler -> service -> repository` boundaries. Existing migrations describe
the domain; corrections and missing persistence use additional migrations so
already migrated installations can upgrade. The legacy cutover must preserve
source records in `legacy_agro_shield` and validate transferred relationships.

PostgreSQL is the persistence choice for sessions and market reports. Keeping
the legacy process-local maps would lose state on restart and produce different
results between server instances. Adding a separate datastore would add an
operational dependency without a demonstrated need.

Learning uses the existing farming types, journeys, messages, and checkpoints.
AI output is generated through a configured provider using the user's saved
context. Completion is based on persisted user actions, rather than invented
progress or a fixed reply. Provider contract tests use explicit test servers;
they do not count as live-provider verification.

## Initial findings

| Area | Evidence at baseline | Required verification |
| --- | --- | --- |
| Password reset | Upgraded handler changes a password without an OTP; legacy has a two-step flow | No change without a valid, unexpired, single-use code |
| Sessions and HTTP | Process-local sessions; legacy middleware omitted | Expiry, revocation, secure cookies, cross-origin write rejection |
| Configuration | Server bypasses `config.Load`; provider settings only partly connected | Invalid settings fail clearly; configured settings reach the services |
| Market reports | Server constructs `repository/memory` implementations | Reports and aggregate data survive a new repository/server instance |
| Learning | Journeys/messages exist; legacy AI providers are absent; ownership checks incomplete | Separate learner experience, real provider call, persisted conversation, access isolation |
| Wallet | Schema/models exist; old wallet service, repository, and pages absent | Preserve and expose compatible balances/history and validate any restored money flows |
| Database | Existing tests do not exercise PostgreSQL migrations/repositories | Fresh and legacy database tests with synthetic records |
| External services | No application credentials are configured in this workspace | Local contract/failure tests; live verification remains separately identified |

## Verification record

Work in progress. Results will be recorded here as the checks run.
