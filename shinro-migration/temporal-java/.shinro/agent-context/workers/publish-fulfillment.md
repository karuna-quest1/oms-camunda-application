# Worker: publish-fulfillment

- Worker id (manifest `source.workerId`): `worker:oms-camunda-quest1:com.oms.workers.OmsWorkers#publishToFulfillment`
- Activity name: `Publish-fulfillment`
- Language/SDK: java/spring-zeebe
- Registration: src/main/java/com/oms/workers/OmsWorkers.java:198 — `@JobWorker(type = JOB_PUBLISH_FULFILLMENT)`
- autoComplete: true

## Classification

- Side-effect class: **read-only-unverified** (confidence 0.55, provenance rule)
- Rationale: no known I/O sink on resolved paths; unresolved callee(s): ObjectMapper.convertValue
- **Requires human confirmation** — treat the translation as a draft.

## Migration guidance

- Coverage: **AUTO+FLAG**
- Suggested retry policy: maxAttempts=2147483647
- flag: unverified-callee: cannot resolve ObjectMapper.convertValue

## Variables

Read:

Written:

## Source

Read the handler source with read_file: `src/main/java/com/oms/workers/OmsWorkers.java` (source repo root).
Call graph: ObjectMapper.convertValue, Logger.info
