# Migration Agent Context

This directory is your reference material for the Camunda 8 → Temporal migration.
Everything is derivable from `raw/findings.json` and `raw/assessment.json` — the files
here are just a navigable, per-entity view. If a markdown file lacks a field you need,
read the raw JSON (e.g. with grep_search) instead of guessing.

## Map

- `overview.md` — what was analyzed (module, languages, counts)
- `tasks.md` — the migration task list. Keep it current with the update_tasks tool.
- `processes/` — one file per BPMN process (1): `order-processing.md`
- `workers/` — one file per job worker (5): `validate-order.md`, `enrich-order.md`, `publish-fulfillment.md`, `update-dashboard.md`, `validate-payment.md`
- `decisions/` — one file per DMN decision (0): none
- `gaps.md` — analysis gaps, breaking downstream consumers, review stops, uncovered processes
- `source-map.md` — every entity → file:line in the source repository
- `raw/` — verbatim findings + assessment JSON

## How to work

1. Read `tasks.md`, pick the first pending process, work ONLY on it.
2. Read its `processes/<id>.md`, then the referenced `workers/<jobType>.md` files,
   then READ THE ACTUAL SOURCE FILES named in source-map.md (read_file on the repo).
3. Write the Temporal Go code into the output project (temporal) with write_file / edit_file.
4. Write conformance tests for the process (testsuite cases from its BPMN flow),
   then run `go build ./... && go test ./...` (run_shell); fix what breaks.
5. Mark the process done with update_tasks once tests are green, then move to the next.

A cluster assessment (BPMN inventory) was available and is reflected here.

## Uncovered processes (manual)

These processes were inventoried in the cluster but have no BPMN model in this repository:
- `account-closure`
- `base-rate-change-broadcast`
- `complaint-handling`
- `credit-bureau-assessment`
- `customer-onboarding-kyc`
- `document-collection`
- `fund-disbursement`
- `fx-settlement`
- `identity-verification`
- `loan-application`
- `payment-instruction`
- `property-valuation-scheduling`
- `rate-change-notification`
- `regulatory-reporting-batch`
- `sanctions-screening`
- `underwriting-review`
