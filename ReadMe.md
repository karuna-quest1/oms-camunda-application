# Order Management System (Camunda 8 / Zeebe)

A **Camunda 8 (BPMN + job workers)** re-implementation of the Temporal + Go OMS
in [`../oms-temporal-suha-quest1`](../oms-temporal-suha-quest1). Same order
lifecycle, same REST surface, same failure/retry behaviour — expressed as a
BPMN process orchestrated by Zeebe instead of a Temporal workflow.

The orchestration lives in the BPMN model
([`models/order-processing.bpmn`](src/main/resources/models/order-processing.bpmn));
the business logic lives in `@JobWorker` beans. One Spring Boot process hosts
both the REST gateway and the workers.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  HTTP Client (curl / frontend)                                   │
└───────────────────────────┬──────────────────────────────────────┘
                            │ REST (:8090)
┌───────────────────────────▼──────────────────────────────────────┐
│  com.oms.api.OrderController  (Spring Boot HTTP gateway)         │
│   POST /orders            → CreateInstance(order-processing)     │
│   POST /orders/correction → PublishMessage(SupportCorrection)    │
│   POST /orders/payment    → PublishMessage(CapturePayment)       │
│   POST /orders/cancel     → PublishMessage(CancelOrder)          │
│   GET  /orders/status     → read dashboard projection            │
└───────────────────────────┬──────────────────────────────────────┘
                            │ Zeebe gRPC (26500)
┌───────────────────────────▼──────────────────────────────────────┐
│  Camunda 8 / Zeebe                                                │
│   Process: order-processing.bpmn                                 │
│   Job workers (com.oms.workers):                                 │
│     validate-order       (CommerceWorkers, rate-bounded)        │
│     validate-payment     (OmsWorkers)                           │
│     enrich-order         (OmsWorkers)                           │
│     update-dashboard     (OmsWorkers → projection)              │
│     publish-fulfillment  (OmsWorkers)                           │
└───────────────────────────┬──────────────────────────────────────┘
                            │ JDBC
┌───────────────────────────▼──────────────────────────────────────┐
│  Dashboard read model (H2 embedded by default, or PostgreSQL)   │
└──────────────────────────────────────────────────────────────────┘
```

### Key files

| File | Responsibility |
|---|---|
| `src/main/resources/models/order-processing.bpmn` | The orchestration: happy path, correction/payment loops, cancel + TTL event subprocesses |
| `com.oms.api.OrderController` | REST gateway; message publishing; HTTP 409 semantics |
| `com.oms.workers.CommerceWorkers` | `validate-order` (dedicated, rate-bounded) |
| `com.oms.workers.OmsWorkers` | `validate-payment`, `enrich-order`, `update-dashboard`, `publish-fulfillment` |
| `com.oms.dashboard.*` | Durable customer read model (projection) |
| `com.oms.process.ProcessConstants` | Message names, job types, error codes, variable names — the model↔code contract |

---

## Setup & Quick Start

### Prerequisites
- JDK 21, Maven 3.9+
- A running Camunda 8 self-managed estate. This repo assumes the sibling
  `../docker-compose-8` estate (Zeebe gRPC on `localhost:26500`, REST on
  `localhost:8088`, Operate on `localhost:8088`).

### Option A — Local development

**1. Start the Camunda 8 estate** (in `../docker-compose-8`):
```bash
docker compose up -d
```

**2. Run the OMS app** (gateway + workers, auto-deploys the BPMN):
```bash
export ZEEBE_GRPC_ADDRESS=http://localhost:26500
export ZEEBE_REST_ADDRESS=http://localhost:8088
export CAMUNDA_AUTH_METHOD=none
mvn spring-boot:run
```
The REST API listens on `http://localhost:8090`. The dashboard read model uses
an embedded H2 database by default (no external DB needed).

### Option B — Docker Compose
Starts the OMS app and joins it to the estate's `camunda` Docker network:
```bash
docker compose up --build
```

### Sample requests
```bash
# Create an order
curl -X POST http://localhost:8090/orders \
  -H "Content-Type: application/json" \
  -d '{"order_id":"ORD-001","customer_id":"CUST-001","items":[{"item_id":"P1","quantity":2,"unit_price":50}],"total_amount":100,"currency":"USD"}'

# Send payment (RRN must be 12 digits; amount must equal total)
curl -X POST http://localhost:8090/orders/payment \
  -H "Content-Type: application/json" \
  -d '{"order_id":"ORD-001","rrn":"123456789012","amount":100,"currency":"USD","payment_method":"CARD"}'

# Check status (served from the projection)
curl "http://localhost:8090/orders/status?order_id=ORD-001"

# Support correction (only valid while AWAITING_CORRECTION)
curl -X POST http://localhost:8090/orders/correction \
  -H "Content-Type: application/json" \
  -d '{"order_id":"ORD-001","items":[{"item_id":"P2","quantity":1,"unit_price":100}]}'

# Cancel
curl -X POST http://localhost:8090/orders/cancel \
  -H "Content-Type: application/json" \
  -d '{"order_id":"ORD-001","reason":"customer request"}'
```

Open **Operate** (in the estate, `http://localhost:8088`) to watch instances.

---

## Workflow flow

```
START
  │
  ▼
Dashboard: ORDER_RECEIVED
  │
  ▼
validate-order  ──INVALID_ORDER──► Dashboard: AWAITING_CORRECTION
  │  (boundary error)                     │
  │                                Await SupportCorrection message
  │                                       │ (loops back to validate-order)
  ▼
Dashboard: PAYMENT_PENDING
  │
  ▼
Await CapturePayment message
  │
  ▼
validate-payment ──INVALID_PAYMENT──► (boundary error loops back to Await CapturePayment)
  │
  ▼
Dashboard: PAYMENT_CAPTURED
  │
  ▼
enrich-order
  │
  ▼
publish-fulfillment
  │
  ▼
Dashboard: FULFILLED ──► END

At ANY point:
  • CancelOrder message → interrupting event subprocess → Dashboard: CANCELLED → terminate
  • 30-day TTL timer   → interrupting event subprocess → Dashboard: EXPIRED   → terminate
```

---

## How the Temporal design maps to Camunda 8

| Temporal concept | Camunda 8 equivalent |
|---|---|
| `OrderProcessingWorkflow` | BPMN process `order-processing` |
| Activity (`ValidateOrderAPI`, …) | Service task + `@JobWorker` |
| Signal (`CancelOrder`) | Zeebe message, `correlationKey = orderId` |
| Update + validator (`CapturePayment`, `SupportCorrection`) | Zeebe message + **API-layer validation** for the HTTP 409 (see below) |
| Query (`GetOrderStatus`) | Read the dashboard projection |
| `workflow.Await(signal, timeout)` | Message intermediate catch event |
| `NonRetryable ApplicationError` → correction loop | Worker throws **BPMN error** → boundary event → loop |
| Retry policy (attempts, backoff) | Service task job `retries` + worker back-pressure (Zeebe applies backoff on failure) |
| 30-day TTL + selector timer | Interrupting **timer start** event subprocess (`PT720H`) |
| Cancel at any stage | Interrupting **message start** event subprocess |
| `ContinueAsNew` at history limit | Not needed — Zeebe streams history to Elasticsearch; dropped, per migration decision |
| `GetVersion` gating | Not needed — Zeebe versions process definitions natively; new instances use the latest, in-flight instances finish on their deployed version |
| Payload AES codec | Not modelled — no PII enters Zeebe (see below); use Zeebe's transport TLS + at-rest encryption in production |
| Customer dashboard (PostgreSQL) | Same idea — a JDBC projection updated by `update-dashboard` |

### Job type ↔ activity mapping

| Job type | Temporal activity | Retry policy (in BPMN) | On business failure |
|---|---|---|---|
| `validate-order` | `ValidateOrderAPI` | ~infinite (`retries` = Integer.MAX) | throw `INVALID_ORDER` → correction loop |
| `validate-payment` | `ValidatePaymentRRN` | 5 | throw `INVALID_PAYMENT` → await new payment |
| `enrich-order` | `EnrichWithPIM` | ~infinite | — |
| `update-dashboard` | `UpdateCustomerDashboard` | best-effort (swallowed, non-fatal) | — |
| `publish-fulfillment` | `PublishToFulfillment` | ~infinite | — |

---

## Design notes & deliberate differences

**HTTP 409 without a validated update.** Temporal used *synchronous validated
Workflow Updates*: the workflow's validator rejected an illegal correction or
payment and the caller got a 409. Zeebe messages have no validating handler, so
`OrderController` reproduces the 409 at the API layer by checking the current
status in the projection before publishing (terminal → 409, wrong state → 409).
This is the one behaviour with no direct Zeebe primitive.

**No PII in the engine.** Only the `orderId` reference and the order/payment
business fields travel through Zeebe process variables. Customer contact details
(email/name) are intended to live solely in the projection, keyed by `orderId` —
so the Temporal payload-encryption codec has no equivalent here (there is no
sensitive payload in the engine to encrypt).

**Correction application.** A `SupportCorrection` carries only the corrected
`items`. Rather than overwrite the whole `order` variable, the controller
publishes them as `correctionItems`; `validate-order` merges them into `order`
on re-entry and clears the flag — the parity of `WorkflowState.ApplyCorrection`.

**Commerce rate limiting.** Temporal capped the Commerce API at 150 RPS via
`TaskQueueActivitiesPerSecond`. Zeebe has no per-task-queue RPS cap; the
`validate-order` worker bounds concurrency via `maxJobsActive` and would need a
rate-limiting gateway inside the worker for a hard RPS ceiling. This is the only
functional gap versus the Temporal implementation.

**Idempotent order creation.** Temporal used
`WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY` keyed on
`order_<id>`. Zeebe has no business-key dedup, so `POST /orders` rejects
(409) a create for an order that already exists in the projection.

---

## Testing

```bash
mvn test
```

`BpmnModelTest` parses and validates the BPMN with the Zeebe model API and
asserts every service task's job type matches a registered worker — catching
model/worker drift without a running engine. For full end-to-end coverage
against a real engine, add `camunda-process-test-java` tests (Testcontainers).
