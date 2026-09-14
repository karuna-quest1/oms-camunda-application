# Gaps & manual work

## Job types declared in BPMN with no worker in this repo

- `arrange-property-access`
- `book-surveyor-slot`
- `build-regulatory-dataset`
- `chase-document`
- `check-identity-bureau`
- `classify-transactions-dmn`
- `confirm-fx-settlement`
- `core-banking-api-call`
- `escalate-sla-breach`
- `find-affected-clients`
- `flag-payment-for-review`
- `flag-product-book`
- `handle-case-sla-breach`
- `lock-fx-rate`
- `manual-ledger-adjustment`
- `notify-customer`
- `notify-ledger-posted`
- `notify-senior-analyst`
- `notify-valuation-appointment`
- `process-cancellation`
- `process-withdrawal`
- `reconcile-fx-ledger`
- `record-supplementary-evidence`
- `release-fund-reservation`
- `rescreen-sanctions-pep`
- `reserve-funds`
- `retry-alternate-rail`
- `screen-sanctions-pep`
- `send-complaint-status-update`
- `send-document-reminder`
- `settle-fx-leg`
- `submit-payment-rail`

## User tasks (become Temporal signals)

- `ut_first_review`
- `ut_manual_id_review`
- `ut_manual_review`
- `ut_manual_upload`
- `ut_second_review`
- `ut_triage`

## Connector job types (must be reimplemented — connectors do not carry over)

- `io.camunda:http-json:1`

## Analysis gaps

- [major] coverage-gap: raw registration markers (6) exceed recognized worker anchors (5) — a worker registration may not have been parsed
- [info] no-test-oracles: no input→output test oracles were extracted; generated code has no dual-run seed from existing tests

## Uncovered processes (in cluster, not in this repo — manual)

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
