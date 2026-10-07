# Worker: validate-payment

- Worker id (manifest `source.workerId`): `worker:oms-camunda-application:com.oms.workers.OmsWorkers#validatePayment`
- Activity name: `Validate-payment`
- Language/SDK: java/spring-zeebe
- Registration: src/main/java/com/oms/workers/OmsWorkers.java:63 — `@JobWorker(type = JOB_VALIDATE_PAYMENT, autoComplete = false)`
- autoComplete: false (explicit completion)

## Classification

- Side-effect class: **read-only** (confidence 0.95, provenance llm)
- Summary: Validates payment amount against order total and throws a BPMN error if invalid, otherwise completes with the payment variable
- Rationale: The code slice shows only in-memory operations: reads variables via job.getVariablesAsMap(), deserializes with ObjectMapper.convertValue, performs a pure validation check (validationError comparison), and logs. No external I/O is visible—no HTTP calls, database writes, or delegation to mutating operations. The call graph confirms only ObjectMapper.convertValue and Logger calls. The worker either throws a BPMN error or completes with the same payment data it received. This is a pure validation worker with no side effects.

## Migration guidance

- Coverage: **ASSISTED**
- Suggested retry policy: maxAttempts=5
- flag: autoComplete=false + throwError -> typed error branch to user-task

## Variables

Read:

Written:

## Engine API calls (these DISAPPEAR — map to Temporal return patterns)

- `newThrowErrorCommand` (success) at src/main/java/com/oms/workers/OmsWorkers.java:78 — hint: NonRetryableApplicationError(type="")
- `newCompleteCommand` (success) at src/main/java/com/oms/workers/OmsWorkers.java:88 — hint: return output, nil

## Source

Read the handler source with read_file: `src/main/java/com/oms/workers/OmsWorkers.java` (source repo root).
Call graph: ObjectMapper.convertValue, Logger.info, Logger.warn
